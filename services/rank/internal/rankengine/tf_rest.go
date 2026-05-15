package rankengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"recsys_go/services/rank/internal/config"
)

// TFPredictor calls TensorFlow Serving REST /v1/models/{name}:predict.
type TFPredictor struct {
	BaseURL       string
	ModelName     string
	SignatureName string
	InputTensor   string
	FeatureDim    int
	OutputName    string // ModelConf OutputName, default predictions
	Client        *http.Client
}

func NewTFPredictor(c config.TFConfig) *TFPredictor {
	if c.BaseURL == "" {
		return nil
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
	return &TFPredictor{
		BaseURL:       strings.TrimRight(strings.TrimSpace(c.BaseURL), "/"),
		ModelName:     c.ModelName,
		SignatureName: sig,
		InputTensor:   in,
		FeatureDim:    dim,
		OutputName:    outName,
		Client: &http.Client{
			Timeout: to,
		},
	}
}

func (p *TFPredictor) Configured() bool {
	return p != nil && p.BaseURL != "" && p.ModelName != ""
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
	url := fmt.Sprintf("%s/v1/models/%s:predict", p.BaseURL, p.ModelName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("tf predict http %d: %s", resp.StatusCode, string(b))
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
