package proxy

import (
	"crypto/tls"
	"errors"
	"sync"
	"time"
)

var errConfigNotFound = errors.New("TLS config not found")

// expireTTL defines the time
const expireTTL = 10 * time.Minute

type cacheEntry struct {
	// cfg defines the TLS config we're caching
	cfg        *tls.Config
	remoteAddr string

	// added holds the time the cfg was added to the cache
	added time.Time
}

type tlsCache struct {
	// configs holds the TLS config for each remote instance
	configs   map[string]cacheEntry
	configsMu sync.Mutex // protects configs

	// nowFn returns the current local time, used during insertion of cache
	// entries. It's a function so we can use it for tests.
	nowFn func() time.Time
}

func newtlsCache() *tlsCache {
	return &tlsCache{
		configs: make(map[string]cacheEntry),
		nowFn:   time.Now,
	}
}

// Add adds the given config and remote address for the given instance name
func (t *tlsCache) Add(instance string, cfg *tls.Config, remoteAddr string) {
	t.configsMu.Lock()
	defer t.configsMu.Unlock()

	t.configs[instance] = cacheEntry{
		cfg:        cfg,
		remoteAddr: remoteAddr,
		added:      t.nowFn(),
	}
}

// Get retrieves the config for the given instance
func (t *tlsCache) Get(instance string) (cacheEntry, error) {
	t.configsMu.Lock()
	defer t.configsMu.Unlock()

	e, ok := t.configs[instance]
	if !ok {
		return cacheEntry{}, errConfigNotFound
	}

	now := time.Now()

	// delete the config if it's expired. This will trigger the user to request
	// another TLS config.
	if e.added.Add(expireTTL).Before(now) {
		delete(t.configs, instance)
		return cacheEntry{}, errConfigNotFound
	}

	return e, nil
}
