package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

const (
	keepAlivePeriod = time.Minute
)

// Cert represents the client certificate key pair in the root certiciate
// authority that the client uses to verify server certificates.
type Cert struct {
	ClientCert tls.Certificate
	CACert     *x509.Certificate
	RemoteAddr string
}

// CertSource is used
type CertSource interface {
	// Cert returns the required certs needed to establish a TLS connection
	// from the client to the server.
	Cert(ctx context.Context, org, db, branch string) (*Cert, error)
}

// Client is responsible for listening to unsecured connections over a TCP
// localhost port and tunneling them securely over a TLS connection to a remote
// database instance defined by its PlanetScale unique branch identifier.
type Client struct {
	remoteAddr     string
	localAddr      string
	instance       string
	maxConnections uint64
	certSource     CertSource

	log *zap.Logger

	// connectionsCounter is used to enforce the optional maxConnections limit
	connectionsCounter uint64

	// configCache contains the TLS certificate chache for each indiviual
	// database
	configCache *tlsCache
}

// Options are the options for creating a new Client.
type Options struct {
	// RemoteAddr defines the server address to tunnel local connections. By
	// default we connect to the remote address given by the CertSource. This
	// option can be used to over write it.
	RemoteAddr string

	// LocalAddr defines the address to listen for new connection
	LocalAddr string

	// Instance defines the remote DB instance to proxy new connection
	Instance string

	// MaxConnections is the maximum number of connections to establish
	// before refusing new connections. 0 means no limit.
	MaxConnections uint64

	// CertSource defines the certificate source to obtain the required TLS
	// certificates for the client and the remote address of the server to
	// connect.
	CertSource CertSource

	// Logger defines which zap.Logger to use. Use it to override the default
	// Development logger . Useful for tests.
	Logger *zap.Logger
}

// NewClient creates a new proxy client instance
func NewClient(opts Options) (*Client, error) {
	c := &Client{
		certSource:  opts.CertSource,
		localAddr:   opts.LocalAddr,
		remoteAddr:  opts.RemoteAddr,
		instance:    opts.Instance,
		configCache: newtlsCache(),
	}

	if opts.Logger != nil {
		c.log = opts.Logger
	} else {
		logger, err := zap.NewDevelopment(
			zap.Fields(zap.String("app", "sql-proxy-client")),
		)
		if err != nil {
			return nil, err
		}
		zap.ReplaceGlobals(logger)
		c.log = logger
	}

	// cache the certs for the given instance(s)
	_, _, err := c.clientCerts(context.Background(), opts.Instance)
	if err != nil {
		c.log.Error("couldn't retrieve TLS certificate for the client", zap.Error(err))
	}

	return c, nil
}

// Conn represents a connection from a client to a specific instance.
type Conn struct {
	Instance string
	Conn     net.Conn
}

// Run runs the proxy. It listens to the configured localhost address and
// proxies the connection over a TLS tunnel to the remote DB instance.
func (c *Client) Run(ctx context.Context) error {
	c.log.Info("ready for new connections")
	l, err := c.getListener()
	if err != nil {
		return fmt.Errorf("error net.Listen: %w", err)
	}
	defer c.log.Sync() // nolint: errcheck

	return c.run(ctx, l)
}

func (c *Client) getListener() (net.Listener, error) {
	if strings.HasPrefix(c.localAddr, "unix://") {
		p := strings.TrimPrefix(c.localAddr, "unix://")
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to remove unix domain socket file %s, error: %s", p, err)
		}
		return net.Listen("unix", p)
	}
	return net.Listen("tcp", c.localAddr)
}

// run is an internal function for testing the Client proxy event loop for
// handling TCP connections
func (c *Client) run(ctx context.Context, l net.Listener) error {
	connSrc := make(chan Conn, 1)
	go func() {
		if err := c.listen(l, connSrc); err != nil {
			c.log.Error("listen to local address", zap.Error(err))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			termTimeout := time.Second * 1
			c.log.Info("received context cancellation, waiting until timeout",
				zap.Duration("timeout", termTimeout))

			err := c.Shutdown(termTimeout)
			if err != nil {
				return fmt.Errorf("error during shutdown: %v", err)
			}
			return nil
		case conn := <-connSrc:
			go func(lc Conn) {
				// TODO(fatih): detach context from parent
				err := c.handleConn(ctx, lc.Conn, lc.Instance)
				if err != nil {
					c.log.Error("error proxying conns", zap.Error(err))
				}
			}(conn)
		}
	}
}

// listen listens to the client's localAddres and sends each incoming
// connections to the given connSrc channel.
func (c *Client) listen(l net.Listener, connSrc chan<- Conn) error {
	c.log.Info("listening remote DB instance",
		zap.String("local_addr", c.localAddr),
		zap.String("instance", c.instance),
	)

	for {
		start := time.Now()
		conn, err := l.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				d := 10*time.Millisecond - time.Since(start)
				if d > 0 {
					time.Sleep(d)
				}
				continue
			}
			l.Close()

			return fmt.Errorf("error in accept for on %v: %w", c.localAddr, err)
		}

		c.log.Info("new connection", zap.String("conn_addr", l.Addr().String()))

		switch clientConn := conn.(type) {
		case *net.TCPConn:
			clientConn.SetKeepAlive(true)                  //nolint: errcheck
			clientConn.SetKeepAlivePeriod(1 * time.Minute) //nolint: errcheck
		}

		connSrc <- Conn{
			Conn:     conn,
			Instance: c.instance,
		}
	}
}

func (c *Client) handleConn(ctx context.Context, conn net.Conn, instance string) error {
	log := c.log.With(zap.String("instance", instance))
	active := atomic.AddUint64(&c.connectionsCounter, 1)

	// Deferred decrement of ConnectionsCounter upon connection closing
	defer atomic.AddUint64(&c.connectionsCounter, ^uint64(0))

	if c.maxConnections > 0 && active > c.maxConnections {
		conn.Close()
		return fmt.Errorf("too many open connections (max %d)", c.maxConnections)
	}

	cfg, remoteAddr, err := c.clientCerts(ctx, instance)
	if err != nil {
		return fmt.Errorf("couldn't retrieve certs for instance: %q: %w", instance, err)
	}

	// TODO(fatih): implement refreshing certs
	// go p.refreshCertAfter(instance, timeToRefresh)

	// overwrite the remote address if the user explicitly set it
	if c.remoteAddr != "" {
		remoteAddr = c.remoteAddr
	}

	c.log.Info("conneting to remote server", zap.String("remote_addr", remoteAddr))

	var d net.Dialer
	remoteConn, err := d.DialContext(ctx, "tcp", remoteAddr)
	if err != nil {
		conn.Close()
		return fmt.Errorf("couldn't connect to %q: %v", remoteAddr, err)
	}

	type setKeepAliver interface {
		SetKeepAlive(keepalive bool) error
		SetKeepAlivePeriod(d time.Duration) error
	}

	if s, ok := conn.(setKeepAliver); ok {
		if err := s.SetKeepAlive(true); err != nil {
			log.Error("couldn't set KeepAlive to true", zap.Error(err))
		} else if err := s.SetKeepAlivePeriod(keepAlivePeriod); err != nil {
			log.Error("couldn't set KeepAlivePeriod", zap.Error(err), zap.Duration("keep_alive_period", keepAlivePeriod))
		}
	} else {
		log.Warn("KeepAlive not supported: long-running tcp connections may be killed by the OS.")
	}

	secureConn := tls.Client(remoteConn, cfg)
	if err := secureConn.Handshake(); err != nil {
		secureConn.Close()
		return fmt.Errorf("couldn't initiate TLS handshake to remote addr: %s", err)
	}

	// Hasta la vista, baby
	copyThenClose(
		secureConn,
		conn,
		"remote connection",
		"local connection on "+conn.LocalAddr().String(),
	)
	return nil
}

// clientCerts returns the TLS configuration needed for the TLS handshake and
// connection
func (c *Client) clientCerts(ctx context.Context, instance string) (*tls.Config, string, error) {
	cacheEntry, err := c.configCache.Get(instance)
	if err == nil {
		c.log.Info("using tls.Config from the cache", zap.String("instance", instance))
		return cacheEntry.cfg, cacheEntry.remoteAddr, nil
	}

	if err != errConfigNotFound {
		return nil, "", err // we don't handle non errConfigNotFound errors
	}

	s := strings.Split(instance, "/")
	if len(s) != 3 {
		return nil, "", fmt.Errorf("instance format is malformed, should be in form organization/dbname/branch, have: %q", instance)
	}

	cert, err := c.certSource.Cert(ctx, s[0], s[1], s[2])
	if err != nil {
		return nil, "", fmt.Errorf("couldn't retrieve certs from cert source: %s", err)
	}

	rootCertPool := x509.NewCertPool()
	rootCertPool.AddCert(cert.CACert)

	cfg := &tls.Config{
		ServerName:   instance,
		Certificates: []tls.Certificate{cert.ClientCert},
		RootCAs:      rootCertPool,
		// Set InsecureSkipVerify to skip the default validation we are
		// replacing. This will not disable VerifyConnection.
		InsecureSkipVerify: true,
		VerifyConnection: func(cs tls.ConnectionState) error {
			// For now, only verify the server's certificate chain.
			// We don't know yet what the server's FQDN will be.
			//
			// 			serverName := cs.ServerName
			// 			commonName := cs.PeerCertificates[0].Subject.CommonName
			// 			if commonName != serverName {
			// 				return fmt.Errorf("invalid certificate name %q, expected %q", commonName, serverName)
			// 			}
			opts := x509.VerifyOptions{
				Roots:         rootCertPool,
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := cs.PeerCertificates[0].Verify(opts)
			return err
		},
	}

	c.log.Info("adding tls.Config to the cache", zap.String("instance", instance))
	c.configCache.Add(instance, cfg, cert.RemoteAddr)
	return cfg, cert.RemoteAddr, nil
}

// Shutdown waits up to a given amount of time for all active connections to
// close. Returns an error if there are still active connections after waiting
// for the whole length of the timeout.
func (c *Client) Shutdown(timeout time.Duration) error {
	term, ticker := time.After(timeout), time.NewTicker(100*time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if atomic.LoadUint64(&c.connectionsCounter) > 0 {
				continue
			}
			c.log.Info("no connections to wait, bailing out")
		case <-term:
		}
		break
	}

	active := atomic.LoadUint64(&c.connectionsCounter)
	if active == 0 {
		return nil
	}
	return fmt.Errorf("%d active connections still exist after waiting for %v", active, timeout)
}

func copyThenClose(remote, local io.ReadWriteCloser, remoteDesc, localDesc string) {
	firstErr := make(chan error, 1)

	go func() {
		readErr, err := myCopy(remote, local)
		select {
		case firstErr <- err:
			if readErr && err == io.EOF {
				zap.L().Info("client closed connection",
					zap.String("local_desc", localDesc))
			} else {
				logError(localDesc, remoteDesc, readErr, err)
			}
			remote.Close()
			local.Close()
		default:
		}
	}()

	readErr, err := myCopy(local, remote)
	select {
	case firstErr <- err:
		if readErr && err == io.EOF {
			zap.L().Info("instance closed connection",
				zap.String("remote_desc", remoteDesc))
		} else {
			logError(remoteDesc, localDesc, readErr, err)
		}
		remote.Close()
		local.Close()
	default:
		// In this case, the other goroutine exited first and already printed its
		// error (and closed the things).
	}
}

func logError(readDesc, writeDesc string, readErr bool, err error) {
	var desc string
	if readErr {
		desc = "reading data from " + readDesc
	} else {
		desc = "writing data to " + writeDesc
	}
	zap.L().Error("copy error", zap.String("desc", desc), zap.Error(err))
}

// myCopy is similar to io.Copy, but reports whether the returned error was due
// to a bad read or write. The returned error will never be nil
func myCopy(dst io.Writer, src io.Reader) (readErr bool, err error) {
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				if err == nil {
					return false, werr
				}
				// Read and write error; just report read error (it happened first).
				return true, err
			}
		}
		if err != nil {
			return true, err
		}
	}
}
