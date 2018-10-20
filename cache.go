package cache

import (
	"time"
	"sync"
	"sync/atomic"
)

type Underlay map[string]*Value

type Value struct {
	data   interface{}
	atime  time.Time
	ttl    time.Duration
}

type Cache struct {
	mu   sync.Mutex
	data Underlay
}


var (
	cache    *Cache
	
	errorC   int32
	successC int32
	
	defaultCleanUpTicker  *time.Ticker
	cleanUpSec time.Duration 
)


func init() {
	u := make(Underlay, 0)
	cache = new(Cache)
	cache.data = u

	defaultCleanUpTicker = time.NewTicker(30 * time.Minute)
	go func() {
		for _ = range defaultCleanUpTicker.C {
			cleanUp()		
		}
	}()
}


//func AddByBytes(k []byte, v interface{}, ttl int) {
//	map[string(k)] - advice official
//
//}

//func GetByBytes(k []byte) interface{} {
//	return nil
//}

func Add(k string, d interface{}, ttl time.Duration) {
	now := time.Now()
	
	v := new(Value)
	v.atime = now
	v.ttl = ttl
	v.data = d
	
	cache.mu.Lock()
	defer cache.mu.Unlock()
	
	cache.data[k] = v
}

func Get(k string) interface{} {
	cache.mu.Lock()
        defer cache.mu.Unlock()
        
        v, ok := cache.data[k]
        if !ok {
        	
        	go inc(&errorC)
        	
        	return nil
	}
	
	inc(&successC)        
	
	cache.data[k].atime = time.Now()
	return v.data		
}


func Size() int {
        cache.mu.Lock()
        defer cache.mu.Unlock()

        return len(cache.data)
}


func Stats() (s, e int32) {
	s = atomic.LoadInt32(&errorC) 
	e = atomic.LoadInt32(&successC)
	return
}


func SetCleanUpTime(t time.Duration) {
	//stop default cleaner
	defaultCleanUpTicker.Stop()
	
	newCleanUpTicker := time.NewTicker(t)
        go func() {
                for _ = range newCleanUpTicker.C {
                        cleanUp()               
                }
        }()
}


func cleanUp() {
	cache.mu.Lock()
        defer cache.mu.Unlock()
	
	var keysToDelete []string
	for k, v := range cache.data {
		if time.Since(v.atime) > v.ttl {
			keysToDelete = append(keysToDelete, k)
		}
	}
	
	for _, k := range keysToDelete {
		delete(cache.data, k)
	}
}


//atomic.AddUint64(&ops, 1)
func inc(c *int32) {
	atomic.AddInt32(c, 1)
}


func dec(c *int32) {
	atomic.AddInt32(c, -1)
}

