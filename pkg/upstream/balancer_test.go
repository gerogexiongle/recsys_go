package upstream

import "testing"

func TestResolveEndpointsPreservesDuplicates(t *testing.T) {
	eps := EndpointsConfig{
		Endpoints: []string{"http://127.0.0.1:18081/", "http://127.0.0.1:18081", "http://127.0.0.1:18081"},
	}.Resolve()
	if len(eps) != 3 {
		t.Fatalf("want 3 duplicate entries, got %v", eps)
	}
}

func TestResolveEndpointsIgnoresBaseURLWhenListSet(t *testing.T) {
	eps := EndpointsConfig{
		BaseURL:   "http://a/",
		Endpoints: []string{"http://b", "http://a"},
	}.Resolve()
	if len(eps) != 2 || eps[0] != "http://b" || eps[1] != "http://a" {
		t.Fatalf("got %v", eps)
	}
}

func TestResolveBaseURLOnly(t *testing.T) {
	eps := EndpointsConfig{BaseURL: "http://a/"}.Resolve()
	if len(eps) != 1 || eps[0] != "http://a" {
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
