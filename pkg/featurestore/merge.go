package featurestore

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type featDoc struct {
	FMSparse []struct {
		K json.RawMessage `json:"k"`
		W float64         `json:"w"`
	} `json:"fm_sparse"`
	TFDense     []float64          `json:"tf_dense"`
	Age         *float64           `json:"age"`
	Gender      *float64           `json:"gender"`
	IncomeWan   *float64           `json:"income_wan"`
	CTR7d       *float64           `json:"ctr_7d"`
	Revenue7d   *float64           `json:"revenue_7d"`
	UserProfile *struct {
		Age    *float64 `json:"age"`
		Gender *float64 `json:"gender"`
	} `json:"user_profile,omitempty"`
	UserFinance *struct {
		IncomeWan *float64 `json:"income_wan"`
	} `json:"user_finance,omitempty"`
	ItemStats *struct {
		CTR7d     *float64 `json:"ctr_7d"`
		Revenue7d *float64 `json:"revenue_7d"`
	} `json:"item_stats,omitempty"`
}

type SparseKV struct {
	Key    int64
	Weight float64
}

func MergeUserItemJSON(user, item []byte) ([]SparseKV, []float64, error) {
	var du, di featDoc
	if len(user) > 0 {
		if err := json.Unmarshal(user, &du); err != nil {
			return nil, nil, fmt.Errorf("user json: %w", err)
		}
	}
	if len(item) > 0 {
		if err := json.Unmarshal(item, &di); err != nil {
			return nil, nil, fmt.Errorf("item json: %w", err)
		}
	}
	coalesceFeatDoc(&du)
	coalesceFeatDoc(&di)
	var out []SparseKV
	for _, e := range du.FMSparse {
		k, err := parseK(e.K)
		if err == nil {
			out = append(out, SparseKV{Key: k, Weight: e.W})
		}
	}
	for _, e := range di.FMSparse {
		k, err := parseK(e.K)
		if err == nil {
			out = append(out, SparseKV{Key: k, Weight: e.W})
		}
	}
	out = appendSemanticSlots(out, &du, &di)
	dense := di.TFDense
	if len(dense) == 0 {
		dense = du.TFDense
	}
	return out, dense, nil
}

func coalesceFeatDoc(d *featDoc) {
	if d.UserProfile != nil {
		if d.Age == nil {
			d.Age = d.UserProfile.Age
		}
		if d.Gender == nil {
			d.Gender = d.UserProfile.Gender
		}
	}
	if d.UserFinance != nil && d.IncomeWan == nil {
		d.IncomeWan = d.UserFinance.IncomeWan
	}
	if d.ItemStats != nil {
		if d.CTR7d == nil {
			d.CTR7d = d.ItemStats.CTR7d
		}
		if d.Revenue7d == nil {
			d.Revenue7d = d.ItemStats.Revenue7d
		}
	}
}

func fmSlotKey(field int) int64 {
	if field <= 0 {
		return 0
	}
	return int64(uint64(uint32(0)) | (uint64(uint32(field)) << 32))
}

func appendSemanticSlots(out []SparseKV, user, item *featDoc) []SparseKV {
	if user != nil {
		if user.Age != nil {
			a := clamp(*user.Age, 0, 100)
			out = append(out, SparseKV{Key: fmSlotKey(1), Weight: a / 100.0})
		}
		if user.Gender != nil {
			out = append(out, SparseKV{Key: fmSlotKey(2), Weight: clamp(*user.Gender, 0, 1)})
		}
		if user.IncomeWan != nil {
			inc := clamp(*user.IncomeWan, 1, 10)
			out = append(out, SparseKV{Key: fmSlotKey(3), Weight: inc / 10.0})
		}
	}
	if item != nil {
		if item.CTR7d != nil {
			out = append(out, SparseKV{Key: fmSlotKey(4), Weight: clamp(*item.CTR7d, 0, 1)})
		}
		if item.Revenue7d != nil {
			rev := *item.Revenue7d
			if rev < 0 {
				rev = 0
			}
			out = append(out, SparseKV{Key: fmSlotKey(5), Weight: rev / 100000.0})
		}
	}
	return out
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func parseK(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 {
		return 0, fmt.Errorf("empty k")
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}
