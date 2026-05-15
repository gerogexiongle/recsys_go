package featurestore

import (
	"context"

	"recsys_go/pkg/recsyskit"
)

type Session struct {
	UserID int64
	User   []byte
	Items  map[int64][]byte
}

// CenterSession: profile per entity + merged filter blobs (item-level).
type CenterSession struct {
	Profile       *Session
	Exposure      map[recsyskit.ItemID]int // from recsysgo:filter:exposure
	FeatureLess   map[int64]struct{}     // from recsysgo:filter:featureless
	LabelByItem   map[int64]string       // from recsysgo:filter:label
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
	expRaw, expMiss, err := st.FilterExposureJSON(ctx)
	if err != nil {
		return nil, err
	}
	if m := ParseExposureJSON(expRaw, expMiss); len(m) > 0 {
		cs.Exposure = make(map[recsyskit.ItemID]int, len(m))
		for id, c := range m {
			cs.Exposure[recsyskit.ItemID(id)] = c
		}
	}
	flRaw, flMiss, err := st.FilterFeatureLessJSON(ctx)
	if err != nil {
		return nil, err
	}
	cs.FeatureLess = ParseFeatureLessSet(flRaw, flMiss)
	lbRaw, lbMiss, err := st.FilterLabelJSON(ctx)
	if err != nil {
		return nil, err
	}
	cs.LabelByItem = ParseLabelMap(lbRaw, lbMiss)
	return cs, nil
}

func (cs *CenterSession) EnrichItems(items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	out := make([]recsyskit.ItemInfo, len(items))
	for i, it := range items {
		out[i] = it
		id := int64(it.ID)
		if cs.FeatureLess != nil {
			if _, drop := cs.FeatureLess[id]; drop {
				if out[i].Extra == nil {
					out[i].Extra = make(map[string]string)
				}
				out[i].Extra["feature_less"] = "1"
			}
		}
		if cs.LabelByItem != nil {
			if lb := cs.LabelByItem[id]; lb != "" {
				if out[i].Extra == nil {
					out[i].Extra = make(map[string]string)
				}
				out[i].Extra["label"] = lb
			}
		}
	}
	return out
}
