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
// When Endpoints is set, order and duplicates are preserved (same host repeated = RR weight / lab test).
// BaseURL is used only when Endpoints is empty (single instance or legacy yaml).
func (c EndpointsConfig) Resolve() []string {
	if len(c.Endpoints) > 0 {
		var out []string
		for _, u := range c.Endpoints {
			u = strings.TrimSpace(u)
			u = strings.TrimRight(u, "/")
			if u != "" {
				out = append(out, u)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if c.BaseURL != "" {
		u := strings.TrimSpace(c.BaseURL)
		u = strings.TrimRight(u, "/")
		if u != "" {
			return []string{u}
		}
	}
	return nil
}
