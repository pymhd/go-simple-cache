package backend

import (
        "time"
)


type Backend interface {
        Get(k string) (interface{}, error)
        Add(k string, v interface{}, ttl time.Duration) error
        Flush() error
        Clean()
}
