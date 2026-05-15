package upstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPDoerRoundRobinAcrossEndpoints(t *testing.T) {
	var c1, c2, c3 atomic.Int32
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c1.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer s1.Close()
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c2.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer s2.Close()
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c3.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer s3.Close()

	doer, err := NewHTTPDoer(EndpointsConfig{
		Endpoints:   []string{s1.URL, s2.URL, s3.URL},
		LoadBalance: "round_robin",
	}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 9; i++ {
		if _, err := doer.PostBytes(context.Background(), "/ping", []byte("{}"), "application/json"); err != nil {
			t.Fatalf("req %d: %v", i, err)
		}
	}
	if c1.Load() != 3 || c2.Load() != 3 || c3.Load() != 3 {
		t.Fatalf("round_robin hits c1=%d c2=%d c3=%d", c1.Load(), c2.Load(), c3.Load())
	}
}

func TestHTTPDoerDuplicateEndpointSameServer(t *testing.T) {
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()
	u := ts.URL

	doer, err := NewHTTPDoer(EndpointsConfig{
		Endpoints:   []string{u, u, u},
		LoadBalance: "round_robin",
	}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(doer.Balancer.All()) != 3 {
		t.Fatalf("want 3 resolved endpoints, got %v", doer.Balancer.All())
	}
	for i := 0; i < 6; i++ {
		if _, err := doer.PostBytes(context.Background(), "/ping", []byte("{}"), "application/json"); err != nil {
			t.Fatalf("req %d: %v", i, err)
		}
	}
	if hits.Load() != 6 {
		t.Fatalf("expected 6 hits on single server, got %d", hits.Load())
	}
}
