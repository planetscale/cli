package proxyutil

import (
	"log/slog"
	"net"

	"github.com/planetscale/psdb/auth"
	"github.com/planetscale/psdbproxy"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"vitess.io/vitess/go/mysql"
)

type Config struct {
	Logger       *zap.Logger
	LocalAddr    string
	UpstreamAddr string
	Username     string
	Password     string
	Database     string
}

// psdbWrapper wraps psdbproxy.Server to implement our Proxy interface
type psdbWrapper struct {
	*psdbproxy.Server
}

func (p *psdbWrapper) Serve(l net.Listener, authMethods ...mysql.AuthMethodDescription) error {
	if len(authMethods) > 0 {
		return p.Server.Serve(l, authMethods[0])
	}
	return p.Server.Serve(l, mysql.CachingSha2Password)
}

func (p *psdbWrapper) Close() {
	p.Server.Shutdown()
}

func New(cfg Config) Proxy {
	server := &psdbproxy.Server{
		Addr:          cfg.LocalAddr,
		Logger:        slog.New(zapslog.NewHandler(cfg.Logger.Core())),
		UpstreamAddr:  cfg.UpstreamAddr,
		Authorization: auth.NewBasicAuth(cfg.Username, cfg.Password),
	}
	return &psdbWrapper{Server: server}
}

// Proxy interface for both MySQL and PostgreSQL proxies
type Proxy interface {
	Serve(net.Listener, ...mysql.AuthMethodDescription) error
	Close()
}

// NewPostgreSQL creates a new PostgreSQL proxy
func NewPostgreSQL(cfg Config) Proxy {
	return NewPostgresProxy(cfg)
}
