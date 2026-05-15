package featurestore

import (
	"context"

	"recsys_go/pkg/recsyskit"
)

// Session holds per-request profile JSON (FM / rank).
type Session struct {
	UserID int64
	User   []byte
	Items  map[int64][]byte
}

// CenterSession adds filter-strategy payloads loaded from separate Redis keys.
type CenterSession struct {
	Profile  *Session
	Exposure map[recsyskit.ItemID]int
	// per-item strategy: key missing in maps => strategy data absent
	FeatureLess map[int64][]byte // raw only when key existed
	Label       map[int64][]byte
}

func LoadSession(ctx context.Context, fetch Fetcher, userID int64, itemIDs []int64) (*Session, error) {
	s := &Session{UserID: userID, Items: make(map[int64][]byte)}
	if fetch == nil || fetch == NoOp {
		return s, nil
	}
	u, err := fetch.UserJSON(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.User = u
	if len(itemIDs) == 0 {
		return s, nil
	}
	if bf, ok := fetch.(BatchFetcher); ok {
		s.Items, err = bf.ItemsJSON(ctx, itemIDs)
		return s, err
	}
	for _, id := range itemIDs {
		b, err := fetch.ItemJSON(ctx, id)
		if err != nil {
			return nil, err
		}
		if len(b) > 0 {
			s.Items[id] = b
		}
	}
	return s, nil
}

// LoadCenterSession loads profile + filter strategy keys (C++ InitUserFeature + game_exposure + map filter flags).
func LoadCenterSession(ctx context.Context, fetch Fetcher, userID int64, itemIDs []int64) (*CenterSession, error) {
	prof, err := LoadSession(ctx, fetch, userID, itemIDs)
	if err != nil {
		return nil, err
	}
	cs := &CenterSession{Profile: prof}
	st, ok := fetch.(StrategyFetcher)
	if !ok || st == nil {
		return cs, nil
	}
	expRaw, expMissing, err := st.UserExposureJSON(ctx, userID)
	if err != nil {
		return nil, err
	}
	if m := ParseExposureJSON(expRaw, expMissing); len(m) > 0 {
		cs.Exposure = make(map[recsyskit.ItemID]int, len(m))
		for id, c := range m {
			cs.Exposure[recsyskit.ItemID(id)] = c
		}
	}
	if len(itemIDs) > 0 {
		cs.FeatureLess, err = st.ItemsFeatureLessJSON(ctx, itemIDs)
		if err != nil {
			return nil, err
		}
		cs.Label, err = st.ItemsLabelJSON(ctx, itemIDs)
		if err != nil {
			return nil, err
		}
	}
	return cs, nil
}

func (cs *CenterSession) EnrichItems(items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	out := make([]recsyskit.ItemInfo, len(items))
	for i, it := range items {
		out[i] = it
		id := int64(it.ID)
		flRaw, flHad := cs.FeatureLess[id]
		if ParseFeatureLessFlag(flRaw, !flHad) {
			if out[i].Extra == nil {
				out[i].Extra = make(map[string]string)
			}
			out[i].Extra["feature_less"] = "1"
		}
		lbRaw, lbHad := cs.Label[id]
		if lb := ParseItemLabel(lbRaw, !lbHad); lb != "" {
			if out[i].Extra == nil {
				out[i].Extra = make(map[string]string)
			}
			out[i].Extra["label"] = lb
		}
	}
	return out
}
