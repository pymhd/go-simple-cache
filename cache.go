package cache

import (
	"fmt"
	"encoding/json"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var (
	errorC   int32
	successC int32

	defaultCleanUpTicker = time.NewTicker(30 * time.Minute)
)

type Underlay map[string]*Value

type Value struct {
	Data       interface{}   `json:"data"`
	Ldata	   []interface{} `json:"ldata"`
	AccessTime time.Time     `json:"attime"`
	TTL        time.Duration `json:"ttl"`
}

type Cache struct {
	mu   sync.Mutex
	Data Underlay `json:"result"`
}

func (cache *Cache) Set(k string, d interface{}, ttl string) {
	now := time.Now()

	v := new(Value)
	v.AccessTime = now
	dur, err := time.ParseDuration(ttl)
	if err != nil {
		dur = time.Duration(365 * 24 * time.Hour)
	}
	v.TTL = dur
	v.Data = d

	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.Data[k] = v
}

func (cache *Cache) Add(k string, d interface{}) {
	cache.mu.Lock()
        defer cache.mu.Unlock()
	
	v, ok  := cache.Data[k]
	if ok {
		v.Ldata = append(v.Ldata, d)
		return
	}
	v = new(Value)
	v.Ldata = []interface{}{d}
	cache.Data[k] = v
}

func (cache *Cache) IsIn(k string, d interface{}) bool {
	cache.mu.Lock()
        defer cache.mu.Unlock()
        
        v, ok  := cache.Data[k]
        if !ok {
        	return false
        }
        for _, data := range v.Ldata {
        	if d == data {
        		return true
        	}
        }
        return false
}

func (cache *Cache) Get(k string) interface{} {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	v, ok := cache.Data[k]
	if !ok {
		go inc(&errorC)
		return nil
	}

	if time.Since(v.AccessTime) > v.TTL {
		go inc(&errorC)
		return nil
	}

	go inc(&successC)

	cache.Data[k].AccessTime = time.Now()
	return v.Data
}

func (cache *Cache) Size() int {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	return len(cache.Data)
}

func (cache *Cache) Stats() (s, e int32) {
	s = atomic.LoadInt32(&successC)
	e = atomic.LoadInt32(&errorC)
	return
}

func (cache *Cache) SetCleanUpTime(t time.Duration) {
	//stop default cleaner
	defaultCleanUpTicker.Stop()

	newCleanUpTicker := time.NewTicker(t)
	go func() {
		for range newCleanUpTicker.C {
			cache.cleanUp()
		}
	}()
}

func (cache *Cache) cleanUp() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	var keysToDelete []string
	for k, v := range cache.Data {
		if time.Since(v.AccessTime) > v.TTL {
			keysToDelete = append(keysToDelete, k)
		}
	}

	for _, k := range keysToDelete {
		delete(cache.Data, k)
	}
}

func (cache *Cache) Save(f string) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	return cache.save(f)
}

func (cache *Cache) save(f string) error {
	out, err := os.Create(f)
	if err != nil {
		return err
	}

	return json.NewEncoder(out).Encode(cache)
}

// atomic.AddUint64(&ops, 1)
func inc(c *int32) {
	atomic.AddInt32(c, 1)
}

func dec(c *int32) {
	atomic.AddInt32(c, -1)
}

func New(file string) *Cache {
	cache := new(Cache)

	in, _ := os.Open(file)
	defer in.Close()

	if err := json.NewDecoder(in).Decode(cache); err != nil {
		//could not load values from file
		u := make(Underlay, 0)
		cache.Data = u
	}
	go func() {
		for range defaultCleanUpTicker.C {
			cache.cleanUp()
		}
	}()
	return cache
}

