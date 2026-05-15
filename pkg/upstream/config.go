package upstream

import "strings"

// EndpointsConfig supports single BaseURL (legacy) or explicit multi-instance list.
// Production: K8s Service single DNS, or list pod IPs / headless service records.
type EndpointsConfig struct {
	BaseURL     string   `json:"BaseURL,optional"`
	Endpoints   []string `json:"Endpoints,optional"`
	LoadBalance string   `json:"LoadBalance,optional"` // round_robin (default) | random
}

// Resolve returns normalized base URLs without trailing slash.
func (c EndpointsConfig) Resolve() []string {
	var raw []string
	if len(c.Endpoints) > 0 {
		raw = append(raw, c.Endpoints...)
	}
	if c.BaseURL != "" {
		raw = append(raw, c.BaseURL)
	}
	seen := make(map[string]struct{}, len(raw))
	var out []string
	for _, u := range raw {
		u = strings.TrimSpace(u)
		u = strings.TrimRight(u, "/")
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}
