package rankengine

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"recsys_go/services/rank/internal/config"
)

func TestTFPredictor_MockServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"predictions": []float64{0.42},
		})
	}))
	defer ts.Close()

	p := NewTFPredictor(config.TFConfig{
		BaseURL:       ts.URL,
		ModelName:     "m1",
		SignatureName: "serving_default",
		InputTensor:   "x",
		FeatureDim:    4,
		TimeoutMs:     1000,
	})
	v, err := p.Predict(context.Background(), []float64{0.1, 0.2, 0.3, 0.4})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(v-0.42) > 1e-6 {
		t.Fatalf("got %v", v)
	}
}
