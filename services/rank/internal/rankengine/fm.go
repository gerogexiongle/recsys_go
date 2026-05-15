package rankengine

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// FMModel mirrors TDPredict::FMModelData + predict_score (FMModel.cpp).
type FMModel struct {
	Factor int
	W0     float64
	W1     map[int64]float64
	W2     map[int64][]float64
}

// LoadFMModel loads the legacy text format; factor comes from config (same as C++ Init factor).
// Each data line: index w1 w2_0 .. w2_{factor-1} [ignored extra columns matching 3*factor+4 rows in some dumps]
func LoadFMModel(path string, cfgFactor int) (*FMModel, error) {
	if path == "" {
		return nil, fmt.Errorf("empty model path")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return nil, fmt.Errorf("fm model: empty")
	}
	head := strings.Fields(sc.Text())
	if len(head) != 4 {
		return nil, fmt.Errorf("fm model first line need 4 fields, got %d: %q", len(head), sc.Text())
	}
	w0, err := strconv.ParseFloat(head[1], 64)
	if err != nil {
		return nil, fmt.Errorf("fm w0: %w", err)
	}
	if cfgFactor <= 0 {
		return nil, fmt.Errorf("fm factor must be > 0")
	}
	m := &FMModel{
		Factor: cfgFactor,
		W0:     w0,
		W1:     make(map[int64]float64),
		W2:     make(map[int64][]float64),
	}
	minTok := 2 + cfgFactor
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		tok := strings.Fields(line)
		if len(tok) < minTok {
			return nil, fmt.Errorf("fm model line need >=%d tokens, got %d", minTok, len(tok))
		}
		idx, err := strconv.ParseInt(tok[0], 10, 64)
		if err != nil {
			return nil, err
		}
		fv := fieldValueFromIndex(idx)
		w1, err := strconv.ParseFloat(tok[1], 64)
		if err != nil {
			return nil, err
		}
		vec := make([]float64, cfgFactor)
		for i := 0; i < cfgFactor; i++ {
			vec[i], err = strconv.ParseFloat(tok[2+i], 64)
			if err != nil {
				return nil, err
			}
		}
		m.W1[fv] = w1
		m.W2[fv] = vec
	}
	return m, nil
}

// fieldValueFromIndex matches FMModelData.h FieldValue packing.
func fieldValueFromIndex(index int64) int64 {
	field := int(index >> 32)
	if field == 0 {
		f := int((index >> 24) & 0xFF)
		v := int(index & 0xFFFFFF)
		return packFieldValue(f, v)
	}
	if field > 0 {
		return index
	}
	return 0
}

// packFieldValue matches TDPredict::FieldValue little-endian {v, f} union read as int64 fv.
func packFieldValue(field, value int) int64 {
	return int64(uint64(uint32(value)) | (uint64(uint32(field)) << 32))
}

// SparseFeature is one non-zero slot (field_value key + weight).
type SparseFeature struct {
	Key    int64
	Weight float64
}

// Predict implements FMModel::predict_feature + predict_score for one item.
func (m *FMModel) Predict(features []SparseFeature) float64 {
	if m == nil {
		return 0
	}
	supScore := m.W0
	supVec := make([]float64, m.Factor)
	supSqr := make([]float64, m.Factor)

	for _, fe := range features {
		if w1, ok := m.W1[fe.Key]; ok {
			supScore += w1 * fe.Weight
		}
		if w2, ok := m.W2[fe.Key]; ok && len(w2) == m.Factor {
			for f := 0; f < m.Factor; f++ {
				d := w2[f] * fe.Weight
				supVec[f] += d
				supSqr[f] += d * d
			}
		}
	}

	result := supScore
	tmp := 0.0
	for f := 0; f < m.Factor; f++ {
		s := supVec[f]
		tmp += s*s - supSqr[f]
	}
	result += 0.5 * tmp
	result = math.Exp(-result)
	return 1.0 / (1.0 + result)
}
