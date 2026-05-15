package rankengine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"recsys_go/pkg/upstream"
	"recsys_go/services/rank/internal/config"
)

// TFPredictor calls TensorFlow Serving REST /v1/models/{name}:predict across multiple docker instances.
// C++ online_map_rank uses gRPC (TFModelGrpc); REST is lower integration cost and enough for item-level predict in lab.
// For batch gRPC + tensor proto, add tf_grpc.go behind TFConfig.Protocol = grpc.
type TFPredictor struct {
	doer          *upstream.HTTPDoer
	ModelName     string
	SignatureName string
	InputTensor   string
	FeatureDim    int
	OutputName    string
}

func NewTFPredictor(c config.TFConfig) (*TFPredictor, error) {
	urls := c.TFEndpoints().Resolve()
	if len(urls) == 0 {
		return nil, nil
	}
	if c.ModelName == "" {
		return nil, nil
	}
	dim := c.FeatureDim
	if dim <= 0 {
		dim = 8
	}
	in := c.InputTensor
	if in == "" {
		in = "inputs"
	}
	sig := c.SignatureName
	if sig == "" {
		sig = "serving_default"
	}
	to := time.Duration(c.TimeoutMs) * time.Millisecond
	if c.TimeoutMs <= 0 {
		to = 1500 * time.Millisecond
	}
	outName := strings.TrimSpace(c.OutputName)
	if outName == "" {
		outName = "predictions"
	}
	doer, err := upstream.NewHTTPDoer(c.TFEndpoints(), to)
	if err != nil {
		return nil, err
	}
	return &TFPredictor{
		doer:          doer,
		ModelName:     c.ModelName,
		SignatureName: sig,
		InputTensor:   in,
		FeatureDim:    dim,
		OutputName:    outName,
	}, nil
}

func (p *TFPredictor) Configured() bool {
	return p != nil && p.doer != nil && p.ModelName != ""
}

// Predict returns the first scalar from predictions (regression head).
func (p *TFPredictor) Predict(ctx context.Context, feature []float64) (float64, error) {
	if !p.Configured() {
		return 0, fmt.Errorf("tf predictor not configured")
	}
	if len(feature) < p.FeatureDim {
		pad := make([]float64, p.FeatureDim)
		copy(pad, feature)
		feature = pad
	} else if len(feature) > p.FeatureDim {
		feature = feature[:p.FeatureDim]
	}

	inst := map[string]interface{}{p.InputTensor: feature}
	body := map[string]interface{}{
		"signature_name": p.SignatureName,
		"instances":      []interface{}{inst},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	path := fmt.Sprintf("/v1/models/%s:predict", p.ModelName)
	b, err := p.doer.PostBytes(ctx, path, raw, "application/json")
	if err != nil {
		return 0, err
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(b, &out); err != nil {
		return 0, err
	}
	rawPred, ok := out[p.OutputName]
	if !ok {
		return 0, fmt.Errorf("tf predict: missing output key %q in %s", p.OutputName, string(b))
	}
	var scalar float64
	if err := json.Unmarshal(rawPred, &scalar); err == nil {
		return scalar, nil
	}
	var one []float64
	if err := json.Unmarshal(rawPred, &one); err == nil && len(one) > 0 {
		return one[0], nil
	}
	var two [][]float64
	if err := json.Unmarshal(rawPred, &two); err == nil && len(two) > 0 && len(two[0]) > 0 {
		return two[0][0], nil
	}
	return 0, fmt.Errorf("tf predict: cannot parse %s: %s", p.OutputName, string(rawPred))
}
