package cache

import (
	"go-simple-cache/backend"
	"sync"
	"time"
)

var (
	defaultStoreDuration   = time.Duration(24 * time.Hour)
	defaultJanitorDuration = time.Duration(60 * time.Second)
)

type Cache struct {
	mu sync.Mutex
	b  backend.Backend
}

func (c *Cache) Get(k string) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.b.Get(k)
}

func (c *Cache) Add(k string, v interface{}, ttl string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dur, err := time.ParseDuration(ttl)
	if err != nil {
		dur = defaultStoreDuration
	}
	return c.b.Add(k, v, dur)
}

func (c *Cache) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.b.Flush()
}

func (c *Cache) clean() {
	c.mu.Lock()
        defer c.mu.Unlock()
        
        c.b.Clean()
}


func New(b backend.Backend) *Cache {
	c := new(Cache)
	c.b = b
	// run self cleanup process (delete expired items)
	jt := time.NewTicker(defaultJanitorDuration)
	go func() {
		for range jt.C {
			c.clean()
		}
	}()
	return c
}
