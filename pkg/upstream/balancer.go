package upstream

import (
	"math/rand"
	"strings"
	"sync/atomic"
	"time"
)

// Balancer picks the next upstream base URL (no path).
type Balancer interface {
	Next() string
	All() []string
}

type roundRobin struct {
	urls []string
	i    uint64
}

func (r *roundRobin) Next() string {
	if len(r.urls) == 0 {
		return ""
	}
	n := atomic.AddUint64(&r.i, 1)
	return r.urls[(n-1)%uint64(len(r.urls))]
}

func (r *roundRobin) All() []string {
	return append([]string(nil), r.urls...)
}

type randomPick struct {
	urls []string
	rng  *rand.Rand
}

func (r *randomPick) Next() string {
	if len(r.urls) == 0 {
		return ""
	}
	return r.urls[r.rng.Intn(len(r.urls))]
}

func (r *randomPick) All() []string {
	return append([]string(nil), r.urls...)
}

// NewBalancer builds LB from resolved endpoints. Empty -> nil.
func NewBalancer(urls []string, policy string) Balancer {
	if len(urls) == 0 {
		return nil
	}
	if len(urls) == 1 {
		return &roundRobin{urls: urls}
	}
	switch strings.TrimSpace(strings.ToLower(policy)) {
	case "random":
		return &randomPick{urls: urls, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	default:
		return &roundRobin{urls: urls}
	}
}
