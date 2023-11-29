package ping

import (
	"cmp"
	"context"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

const baseDomain = ".connect.psdb.cloud"

func PingCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		timeout     time.Duration
		concurrency uint8
	}

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Ping public PlanetScale database endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			// XXX: explicitly use a new client that doesn't use authentication
			client, err := ps.NewClient(
				ps.WithBaseURL(ps.DefaultBaseURL),
			)
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress("Fetching regions...")
			defer end()

			regions, err := client.Regions.List(ctx, &ps.ListRegionsRequest{})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			end()

			// set of unique providers for the optimized endpoints
			providers := make(map[string]struct{})
			for _, r := range regions {
				providers[strings.ToLower(r.Provider)] = struct{}{}
			}

			endpoints := make([]string, 0, len(providers)+len(regions))
			for p := range providers {
				endpoints = append(endpoints, p)
			}
			for _, r := range regions {
				endpoints = append(endpoints, r.Slug)
			}

			end = ch.Printer.PrintProgress("Pinging endpoints...")
			defer end()

			results := pingEndpoints(
				ctx,
				endpoints,
				flags.concurrency,
				flags.timeout,
			)
			end()

			return ch.Printer.PrintResource(toResultsTable(results, providers))
		},
	}

	cmd.PersistentFlags().DurationVar(&flags.timeout, "timeout",
		5*time.Second, "Timeout for a ping to succeed.")
	cmd.PersistentFlags().Uint8Var(&flags.concurrency, "concurrency",
		8, "Number of concurrent pings.")

	return cmd
}

type pingResult struct {
	key string
	d   time.Duration
	err error
}

type Result struct {
	Endpoint string `header:"endpoint" json:"endpoint"`
	Latency  string `header:"latency" json:"latency"`
	Type     string `header:"type" json:"type"`
}

func directOrOptimized(key string, providers map[string]struct{}) string {
	if _, ok := providers[key]; ok {
		return "optimized"
	}
	return "direct"
}

func toResultsTable(results []pingResult, providers map[string]struct{}) []*Result {
	rs := make([]*Result, 0, len(results))
	for _, r := range results {
		row := &Result{
			Endpoint: makeHostname(r.key),
			Type:     directOrOptimized(r.key, providers),
		}
		if r.err == nil {
			row.Latency = r.d.Truncate(100 * time.Microsecond).String()
		} else {
			row.Latency = "---"
		}

		rs = append(rs, row)
	}
	return rs
}

func makeHostname(subdomain string) string {
	return subdomain + baseDomain
}

func pingEndpoints(ctx context.Context, eps []string, concurrency uint8, timeout time.Duration) []pingResult {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		sem     = make(chan struct{}, concurrency)
		results = make([]pingResult, 0, len(eps))
	)

	defer close(sem)

	for _, ep := range eps {
		ep := ep
		wg.Add(1)
		select {
		case <-ctx.Done():
			return results
		case sem <- struct{}{}:
			go func() {
				d, err := pingEndpoint(ctx, makeHostname(ep), timeout)
				// XXX: on failures, set the duration to the timeout so they are sorted last
				if err != nil {
					d = timeout
				}

				mu.Lock()
				results = append(results, pingResult{ep, d, err})
				mu.Unlock()
				wg.Done()
				<-sem
			}()
		}
	}

	wg.Wait()
	slices.SortFunc(results, func(a, b pingResult) int { return cmp.Compare(a.d, b.d) })
	return results
}

func pingEndpoint(ctx context.Context, hostname string, timeout time.Duration) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// make hostname a FQDN if not already
	if hostname[len(hostname)-1] != '.' {
		hostname = hostname + "."
	}

	// separately look up DNS so that's separate from our connection time
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", hostname)
	if err != nil {
		return 0, err
	}
	if len(addrs) == 0 {
		return 0, fmt.Errorf("unable to resolve addr: %s", hostname)
	}

	// explicitly time to establish a TCP connection
	var d net.Dialer
	start := time.Now()
	conn, err := d.DialContext(ctx, "tcp", netip.AddrPortFrom(addrs[0], 443).String())
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return time.Since(start), nil
}
