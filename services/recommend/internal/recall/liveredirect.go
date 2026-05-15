package recall

import (
	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
)

// liveRedirectFromUser returns real-time redirect maps (C++ GetLiveRedirectSampleData).
func liveRedirectFromUser(userJSON []byte, n int) ([]recsyskit.ItemInfo, bool) {
	items := featurestore.ParseLiveRedirectItems(userJSON, n)
	if len(items) == 0 {
		return nil, false
	}
	return items, true
}
