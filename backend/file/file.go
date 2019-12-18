package file

import (
	"os"
	"fmt"
	"time"
	"encoding/json"
)

type underlay map[string]*Value


type Value struct {
	TTL   time.Duration `json:"ttl"`
	Atime time.Time     `json:"atime"`
	V     interface{}   `json:"v"`
}

type FileBackend struct {
	path string
	Data *underlay `json:"data"`
}

func (fb FileBackend) Get(k string) (interface{}, error) {
	v, ok := (*fb.Data)[k]
	
	// if key does not exist in underlay map
	if !ok {
		return nil, fmt.Errorf("item does not exist")
	}

	// if value was created or accessed later then TTL for this obj
	if time.Since(v.Atime) > v.TTL {
		return nil, fmt.Errorf("item expired")
	}

	// set atime to now()
	(*fb.Data)[k].Atime = time.Now()
	return v.V, nil
}

func (fb FileBackend) Add(k string, i interface{}, ttl time.Duration) error {
	v, ok := (*fb.Data)[k]
	
	if !ok {
		// this is new item in cache
		v = new(Value)
		(*fb.Data)[k] = v
	}
	v.Atime = time.Now()
        v.TTL = ttl
        v.V = i
        
        return nil
}

func (fb FileBackend) Flush() error {
	// in case of in memory only backend
	if fb.path == "" {
		return nil
	}
	// write to disk
	o, err := os.Create(fb.path)
	if err != nil {
                return err
        }
        return json.NewEncoder(o).Encode(fb)
}


func (fb FileBackend) Clean() {
	var expired []string
	for k, v := range *fb.Data {
		if time.Since(v.Atime) > v.TTL {
                        expired = append(expired, k)
		}
	}
	
	for _, v := range expired {
                delete(*fb.Data, v)
        }
}

func NewBackend(path string) (FileBackend, error) {
	b := FileBackend{}
        b.path = path
        
	in, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return b, err
	}		
	defer in.Close()
	
	if err := json.NewDecoder(in).Decode(&b); err != nil {
		// empty file
		data := make(underlay)
		b.Data = &data
	}
	
	return b, nil
}

func NewBackendDummy() FileBackend {
        b := FileBackend{}
        
        data := make(underlay)
        b.Data = &data
        
        return b
}
