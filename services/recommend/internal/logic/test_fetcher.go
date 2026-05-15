package logic

import (
	"context"

	"recsys_go/pkg/featurestore"
)

// demoTestFetcher supplies E2E-aligned exposure and item portraits for unit tests (no Redis).
type demoTestFetcher struct {
	featurestore.NoOpFetcher
}

func (demoTestFetcher) FilterExposureJSON(context.Context) ([]byte, bool, error) {
	return []byte(`{"910005":15}`), false, nil
}

func (demoTestFetcher) ItemsJSON(_ context.Context, ids []int64) (map[int64][]byte, error) {
	m := make(map[int64][]byte, len(ids))
	for _, id := range ids {
		if id == 910009 {
			continue
		}
		m[id] = []byte(`{}`)
	}
	return m, nil
}
