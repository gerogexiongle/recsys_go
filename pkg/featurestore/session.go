package featurestore

import (
	"context"

	"recsys_go/pkg/recsyskit"
)

// Session holds per-request user + item JSON (C++ InitUserFeature + InitMapFeature).
type Session struct {
	UserID int64
	User   []byte
	Items  map[int64][]byte
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

func (s *Session) ExposureMap() map[recsyskit.ItemID]int {
	if s == nil {
		return nil
	}
	raw := ParseUserExposure(s.User)
	if len(raw) == 0 {
		return nil
	}
	out := make(map[recsyskit.ItemID]int, len(raw))
	for id, c := range raw {
		out[recsyskit.ItemID(id)] = c
	}
	return out
}

func (s *Session) EnrichItems(items []recsyskit.ItemInfo) []recsyskit.ItemInfo {
	if s == nil || len(s.Items) == 0 {
		return items
	}
	out := make([]recsyskit.ItemInfo, len(items))
	for i, it := range items {
		out[i] = it
		if extra := ParseItemExtra(s.Items[int64(it.ID)]); len(extra) > 0 {
			if out[i].Extra == nil {
				out[i].Extra = make(map[string]string)
			}
			for k, v := range extra {
				out[i].Extra[k] = v
			}
		}
	}
	return out
}
