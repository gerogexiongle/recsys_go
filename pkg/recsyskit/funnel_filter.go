package recsyskit

// ApplyFilterPolicies runs ordered policies (exposure_backoff, ...).
func ApplyFilterPolicies(rctx RequestContext, policies []FilterPolicy, items []ItemInfo) []ItemInfo {
	if len(policies) == 0 || len(items) == 0 {
		return items
	}
	out := items
	for _, p := range policies {
		switch p.Type {
		case "exposure_backoff":
			if p.BackoffAfter <= 0 || rctx.Exposure == nil {
				continue
			}
			out = filterExposureBackoff(out, rctx.Exposure, p.BackoffAfter)
		default:
			continue
		}
	}
	return out
}

func filterExposureBackoff(items []ItemInfo, exposure map[ItemID]int, after int) []ItemInfo {
	var out []ItemInfo
	for _, it := range items {
		if exposure[it.ID] >= after {
			continue
		}
		out = append(out, it)
	}
	return out
}

// ApplyShowControl caps final list length (展控).
func ApplyShowControl(cfg ShowControlCfg, items []ItemInfo) []ItemInfo {
	if cfg.MaxItems <= 0 || len(items) <= cfg.MaxItems {
		return items
	}
	return items[:cfg.MaxItems]
}
