package ping

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/constraints"
)

const baseDomain = ".connect.psdb.cloud"

func ensurePositive[T constraints.Integer](f *pflag.Flag, v T) error {
	if v < 1 {
		var flags string
		if f.Shorthand != "" {
			flags = fmt.Sprintf("-%s, --%s", f.Shorthand, f.Name)
		} else {
			flags = fmt.Sprintf("--%s", f.Name)
		}
		return fmt.Errorf(
			"invalid argument \"%d\" for \"%s\" flag: must be greater than 0",
			v,
			flags,
		)
	}
	return nil
}

func ensureBetween[T constraints.Integer](f *pflag.Flag, v, min, max T) error {
	if v < min || v > max {
		var flags string
		if f.Shorthand != "" {
			flags = fmt.Sprintf("-%s, --%s", f.Shorthand, f.Name)
		} else {
			flags = fmt.Sprintf("--%s", f.Name)
		}
		return fmt.Errorf(
			"invalid argument \"%d\" for \"%s\" flag: must be between %d and %d",
			v,
			flags,
			min, max,
		)
	}
	return nil
}

func PingCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		timeout     time.Duration
		concurrency uint8
		count       uint8
		provider    string
	}

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Ping public PlanetScale database endpoints",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := ensureBetween(cmd.PersistentFlags().Lookup("count"), flags.count, 1, 20); err != nil {
				return err
			}
			if err := ensurePositive(cmd.PersistentFlags().Lookup("concurrency"), flags.concurrency); err != nil {
				return err
			}
			return nil
		},
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

			rd := processRegions(flags.provider, regions)

			end = ch.Printer.PrintProgress("Pinging endpoints...")
			defer end()

			results := pingEndpoints(
				ctx,
				rd.Endpoints,
				flags.concurrency,
				flags.count,
				flags.timeout,
			)
			end()

			return ch.Printer.PrintResource(toResultsTable(
				ctx, results, rd, flags.timeout,
			))
		},
	}

	cmd.PersistentFlags().DurationVar(&flags.timeout, "timeout",
		5*time.Second, "Timeout for each ping to succeed.")
	cmd.PersistentFlags().Uint8Var(&flags.concurrency, "concurrency",
		8, "Number of concurrent pings.")
	cmd.PersistentFlags().Uint8VarP(&flags.count, "count", "n", 10, "Number of pings")
	cmd.PersistentFlags().StringVarP(&flags.provider, "provider", "p", "", `Only ping endpoints for the specified infrastructure provider (options: "aws", "gcp").`)

	return cmd
}

type pingResult struct {
	key string
	d   time.Duration
	err error
}

type Result struct {
	Name     string `header:"name" json:"name"`
	Latency  string `header:"latency" json:"latency"`
	Endpoint string `header:"endpoint" json:"endpoint"`
	Type     string `header:"type" json:"type"`
}

func directOrOptimized(key string, providers map[string]struct{}) string {
	if _, ok := providers[key]; ok {
		return "optimized"
	}
	return "direct"
}

func getDisplayName(ctx context.Context, key string, regions map[string]*ps.Region, timeout time.Duration) string {
	if r, ok := regions[key]; ok {
		return r.Name
	}

	// if we don't map to a region, we need to attempt to query and lookup
	// what an endpoint is resolving to. This is needed for the optimized endpoints
	// which may point at the closest region to the caller.
	if key, err := lookupRegionSlug(ctx, key, timeout); err == nil {
		if r, ok := regions[key]; ok {
			return r.Name
		}
	}

	return "<unknown>"
}

func lookupRegionSlug(ctx context.Context, key string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "HEAD", "https://"+makeHostname(key), nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// XXX: we identify ourselves through the Server header
	// Formatted as:
	//  Server: gateway/{region}/{sha}
	// We are looking to extract the middle region bit
	server := resp.Header.Get("Server")
	bits := strings.SplitN(server, "/", 3)
	if len(bits) != 3 {
		return "", errors.New("server header was malformed")
	}
	return bits[1], nil
}

func toResultsTable(ctx context.Context, results []pingResult, rd regionData, timeout time.Duration) []*Result {
	rs := make([]*Result, 0, len(results))
	for _, r := range results {
		row := &Result{
			Name:     getDisplayName(ctx, r.key, rd.RegionsBySlug, timeout*2),
			Endpoint: makeHostname(r.key),
			Type:     directOrOptimized(r.key, rd.Providers),
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

type regionData struct {
	// Set of unique providers for the optimized endpoints.
	Providers map[string]struct{}

	// Map of each region by its slug.
	RegionsBySlug map[string]*ps.Region

	// The final list of endpoints to ping.
	Endpoints []string
}

// processRegions processes a list of PlanetScale regions into data to begin the
// ping process. If provider is not empty, it is used as a filter to match
// infrastructure for that provider.
func processRegions(provider string, regions []*ps.Region) regionData {
	var (
		providers     = make(map[string]struct{})
		regionsBySlug = make(map[string]*ps.Region)
		endpoints     []string
	)

	for _, r := range regions {
		providers[strings.ToLower(r.Provider)] = struct{}{}
		regionsBySlug[r.Slug] = r
	}

	// The user may have typed "aws" or "AWS"; be lax in comparing the provider
	// strings.
	for p := range providers {
		if provider != "" && !strings.EqualFold(p, provider) {
			continue
		}

		endpoints = append(endpoints, p)
	}

	for _, r := range regions {
		if provider != "" && !strings.EqualFold(r.Provider, provider) {
			continue
		}

		endpoints = append(endpoints, r.Slug)
	}

	sort.Strings(endpoints)

	return regionData{
		Providers:     providers,
		RegionsBySlug: regionsBySlug,
		Endpoints:     endpoints,
	}
}

func pingEndpoints(ctx context.Context, eps []string, concurrency, count uint8, timeout time.Duration) []pingResult {
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
				var sum time.Duration
				var resultErr error
				for i := 0; i < int(count); i++ {
					d, err := pingEndpoint(ctx, makeHostname(ep), timeout)
					// XXX: on failures, set the duration to the timeout so they are sorted last
					if err != nil {
						resultErr = err
						d = timeout
					}
					sum += d
				}

				mu.Lock()
				results = append(results, pingResult{
					key: ep,
					d:   sum / time.Duration(count),
					err: resultErr,
				})
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
