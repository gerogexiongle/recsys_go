package centerconfig

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"recsys_go/pkg/featurestore"
	"recsys_go/pkg/recsyskit"
)

const homogeneityRecallType = "HomogeneityContent"

// ApplyShowStrategiesWithExclusive runs show control after rank (C++ ShowControl::CallShowControl).
func ApplyShowStrategiesWithExclusive(ctx context.Context, feat featurestore.Fetcher, items []recsyskit.ItemInfo, exclusive recsyskit.ExclusivePool, strategies []ShowStrategy) []recsyskit.ItemInfo {
	out := items
	for _, st := range strategies {
		switch st.ShowControlType {
		case "ScoreControl":
			out = applyScoreControl(out, st)
		case "HomogenContent":
			out = applyHomogenContent(ctx, feat, out, st.TopNShowControl)
		case "MMRRearrange":
			out = applyMMRRearrange(out, st)
		case "ForcedInsert":
			out = applyForcedInsertExclusive(out, exclusive, st)
		}
	}
	return out
}

func applyHomogenContent(ctx context.Context, feat featurestore.Fetcher, items []recsyskit.ItemInfo, topN int) []recsyskit.ItemInfo {
	if topN <= 0 || len(items) == 0 || feat == nil || feat == featurestore.NoOp {
		return applyHomogenCap(items, topN)
	}
	hf, ok := feat.(featurestore.HomogenFetcher)
	if !ok {
		return applyHomogenCap(items, topN)
	}
	raw, miss, err := hf.HomogenExchangeJSON(ctx)
	if err != nil || miss {
		return applyHomogenCap(items, topN)
	}
	exchange := featurestore.ParseHomogenExchange(raw, miss)
	if len(exchange) == 0 {
		return applyHomogenCap(items, topN)
	}
	n := topN
	if n <= 0 || n > len(items) {
		n = len(items)
	}
	seen := make(map[recsyskit.ItemID]struct{}, len(items)+n)
	out := make([]recsyskit.ItemInfo, 0, len(items))
	for i := 0; i < n; i++ {
		it := items[i]
		if alt, ok := exchange[int64(it.ID)]; ok && alt > 0 && alt != int64(it.ID) {
			altID := recsyskit.ItemID(alt)
			if _, dup := seen[altID]; !dup {
				out = append(out, recsyskit.ItemInfo{ID: altID, RecallType: homogeneityRecallType, Score: it.Score})
				seen[altID] = struct{}{}
				continue
			}
		}
		if _, dup := seen[it.ID]; !dup {
			out = append(out, it)
			seen[it.ID] = struct{}{}
		}
	}
	for i := n; i < len(items); i++ {
		it := items[i]
		if _, dup := seen[it.ID]; dup {
			continue
		}
		out = append(out, it)
		seen[it.ID] = struct{}{}
	}
	return out
}

func applyForcedInsertExclusive(items []recsyskit.ItemInfo, exclusive recsyskit.ExclusivePool, st ShowStrategy) []recsyskit.ItemInfo {
	if len(st.ForcedInsert) == 0 {
		return items
	}
	var insertQueue []recsyskit.ItemInfo
	for _, rule := range st.ForcedInsert {
		src := exclusive[rule.RecallType]
		if len(src) == 0 {
			src = pickFromMainByRecallType(items, rule.RecallType)
		}
		insertQueue = append(insertQueue, pickForcedInsert(src, rule)...)
	}
	if len(insertQueue) == 0 {
		return items
	}
	if st.PageSize > 0 && st.PageInsertCount > 0 && st.PageNum > 0 {
		out := applyForcedInsertPaged(items, insertQueue, st)
		if len(out) > 0 {
			return out
		}
	}
	return prependUnique(items, insertQueue)
}

func pickFromMainByRecallType(items []recsyskit.ItemInfo, recallType string) []recsyskit.ItemInfo {
	var out []recsyskit.ItemInfo
	for _, it := range items {
		if it.RecallType == recallType {
			out = append(out, it)
		}
	}
	return out
}

func pickForcedInsert(src []recsyskit.ItemInfo, rule ForcedInsertRule) []recsyskit.ItemInfo {
	if len(src) == 0 || rule.ForcedInsertCount <= 0 {
		return nil
	}
	n := rule.ForcedInsertCount
	if n > len(src) {
		n = len(src)
	}
	method := strings.TrimSpace(rule.ExtractMethod)
	if method == "" {
		method = "TopNOrder"
	}
	switch method {
	case "TopNRand":
		shuffled := append([]recsyskit.ItemInfo(nil), src...)
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		return shuffled[:n]
	default:
		return src[:n]
	}
}

func applyForcedInsertPaged(ori []recsyskit.ItemInfo, inserts []recsyskit.ItemInfo, st ShowStrategy) []recsyskit.ItemInfo {
	pageSize := st.PageSize
	pageInsert := st.PageInsertCount
	pageOri := pageSize - pageInsert
	if pageOri <= 0 {
		return prependUnique(ori, inserts)
	}
	total := len(ori)
	actualPages := st.PageNum
	if maxP := total / pageSize; maxP < actualPages {
		actualPages = maxP
	}
	var out []recsyskit.ItemInfo
	insIdx := 0
	for page := 0; page < actualPages && insIdx < len(inserts); page++ {
		baseOff := page * pageOri
		pageList := make([]recsyskit.ItemInfo, 0, pageSize)
		for i := 0; i < pageOri && baseOff+i < total; i++ {
			pageList = append(pageList, ori[baseOff+i])
		}
		for k := 0; k < pageInsert && insIdx < len(inserts); k++ {
			pageList = append(pageList, inserts[insIdx])
			insIdx++
		}
		out = append(out, pageList...)
	}
	if baseOff := actualPages * pageOri; baseOff < total {
		out = append(out, ori[baseOff:]...)
	}
	return dedupeKeepFirst(out)
}

func prependUnique(items, prefix []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	seen := make(map[recsyskit.ItemID]struct{}, len(items)+len(prefix))
	var out []recsyskit.ItemInfo
	for _, it := range prefix {
		if _, ok := seen[it.ID]; ok {
			continue
		}
		seen[it.ID] = struct{}{}
		out = append(out, it)
	}
	for _, it := range items {
		if _, ok := seen[it.ID]; ok {
			continue
		}
		seen[it.ID] = struct{}{}
		out = append(out, it)
	}
	return out
}

func dedupeKeepFirst(items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	seen := make(map[recsyskit.ItemID]struct{}, len(items))
	var out []recsyskit.ItemInfo
	for _, it := range items {
		if _, ok := seen[it.ID]; ok {
			continue
		}
		seen[it.ID] = struct{}{}
		out = append(out, it)
	}
	return out
}
