package rankengine

import (
	"fmt"
	"hash/fnv"
)

// BuildPlaceholderSparse builds deterministic sparse keys until real FMFeature / Redis wiring exists.
func BuildPlaceholderSparse(userID, itemID int64, trans *FMTrans) []SparseFeature {
	if trans == nil || len(trans.FieldTrans) == 0 {
		return []SparseFeature{{Key: crossFeatureKey(userID, itemID), Weight: 1}}
	}
	out := make([]SparseFeature, 0, 32)
	for field, cnt := range trans.FieldTrans {
		for i := 0; i < cnt; i++ {
			h := fnv.New64a()
			_, _ = fmt.Fprintf(h, "%d:%d:%d:%d", userID, itemID, field, i)
			v := int64(h.Sum64() & 0xffffffff)
			if field > 0 {
				out = append(out, SparseFeature{Key: (int64(field) << 32) | v, Weight: 1})
			} else {
				fv := int((v >> 24) & 0xFF)
				vv := int(v & 0xffffff)
				out = append(out, SparseFeature{Key: packFieldValue(fv, vv), Weight: 1})
			}
		}
	}
	return out
}

func crossFeatureKey(userID, itemID int64) int64 {
	h := fnv.New64a()
	_, _ = fmt.Fprintf(h, "u:%d:i:%d", userID, itemID)
	v := int64(h.Sum64() & 0x0fffffff)
	return (int64(1) << 32) | v
}
