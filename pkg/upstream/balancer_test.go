package upstream

import "testing"

func TestResolveEndpointsDedup(t *testing.T) {
	eps := EndpointsConfig{
		BaseURL:   "http://a/",
		Endpoints: []string{"http://b", "http://a"},
	}.Resolve()
	if len(eps) != 2 {
		t.Fatalf("got %v", eps)
	}
}

func TestRoundRobin(t *testing.T) {
	b := NewBalancer([]string{"http://1", "http://2", "http://3"}, "round_robin")
	seen := map[string]int{}
	for i := 0; i < 9; i++ {
		seen[b.Next()]++
	}
	for _, u := range []string{"http://1", "http://2", "http://3"} {
		if seen[u] != 3 {
			t.Fatalf("rr %v", seen)
		}
	}
}
