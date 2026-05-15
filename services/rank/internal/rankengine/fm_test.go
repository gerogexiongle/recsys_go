package rankengine

import (
	"math"
	"testing"
)

func TestLoadFMModelToy(t *testing.T) {
	m, err := LoadFMModel("testdata/fm_toy.txt", 2)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(m.W0-0.1) > 1e-6 {
		t.Fatalf("w0 %v", m.W0)
	}
	sc := m.Predict([]SparseFeature{{Key: 4294967297, Weight: 1}})
	if sc <= 0 || sc >= 1 {
		t.Fatalf("score out of (0,1): %v", sc)
	}
}
