package main
//package cache

import (
	"os"
	"fmt"
	"time"
	"sync"
	"sync/atomic"
	"encoding/json"
)

var (
	errorC   int32
	successC int32

	defaultCleanUpTicker *time.Ticker = time.NewTicker(30 * time.Minute)
)

type Underlay map[string]*Value

type Value struct {
	Data  interface{}   `json:"data"`
	AccessTime time.Time     `json:"attime"`
	TTL   time.Duration `json:"ttl"`
}

type Cache struct {
	mu   sync.Mutex
	Data Underlay `json:"result"`
}

func (cache *Cache) Add(k string, d interface{}, ttl string)  {
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

func (cache *Cache) Get(k string) interface{} {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	v, ok := cache.Data[k]
	if !ok {

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
	s = atomic.LoadInt32(&errorC)
	e = atomic.LoadInt32(&successC)
	return
}

func (cache *Cache) SetCleanUpTime(t time.Duration) {
	//stop default cleaner
	defaultCleanUpTicker.Stop()

	newCleanUpTicker := time.NewTicker(t)
	go func() {
		for _ = range newCleanUpTicker.C {
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
		for _ = range defaultCleanUpTicker.C {
			cache.cleanUp()
		}
	}()
	return cache
}

func (cache *Cache) Save(f string) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	
	return cache.save(f)
}
func  (cache *Cache) save(f string) error {
	out, err := os.Create(f)
	if err != nil {
		return err
	}
	
	return json.NewEncoder(out).Encode(cache)
}

//atomic.AddUint64(&ops, 1)
func inc(c *int32) {
	atomic.AddInt32(c, 1)
}

func dec(c *int32) {
	atomic.AddInt32(c, -1)
}


func main() {
	c := New("txt")
	
	c.Add("test", 5, "60s")
	c.Add("huest", 100, "60s")
	
	//e := c.Save("txt")
	fmt.Println(c.Data["test"])
}

