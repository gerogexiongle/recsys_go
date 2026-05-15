package config

import "github.com/zeromicro/go-zero/rest"

// FMConfig loads the same text format as legacy TDPredict FMModelData (see online_map_rank FMModelData.h).
type FMConfig struct {
	Factor    int    `json:"Factor,optional"`
	ModelPath string `json:"ModelPath,optional"`
	TransPath string `json:"TransPath,optional"`
}

// TFConfig calls TensorFlow Serving HTTP predict API (docker 8501 REST).
type TFConfig struct {
	BaseURL       string `json:"BaseURL,optional"`
	ModelName     string `json:"ModelName,optional"`
	SignatureName string `json:"SignatureName,optional"`
	InputTensor   string `json:"InputTensor,optional"`
	FeatureDim    int    `json:"FeatureDim,optional"`
	TimeoutMs     int    `json:"TimeoutMs,optional"`
	// OutputName matches ModelConf TFModel.OutputName (e.g. predictions); default predictions.
	OutputName string `json:"OutputName,optional"`
}

// RankEngineConfig mirrors C++ RankCalc: PreRank (FM) → Rank (TF) → ReRank (optional / mock).
type RankEngineConfig struct {
	Mode string `json:"Mode,optional"` // mock | pipeline — pipeline uses FM/TF when paths set, else falls back to mock for that stage.

	FM FMConfig `json:"FM,optional"`

	TFServing TFConfig `json:"TFServing,optional"`

	PreRankTrunc int `json:"PreRankTrunc,optional"` // 0 = no extra truncation beyond ret_count
	RankTrunc    int `json:"RankTrunc,optional"`    // align with C++ TruncCount before TF, 0 = disabled
}

// FeatureRedis configures JSON STRING keys for user/item features (open-source default: CN test Redis host).
type FeatureRedis struct {
	Disabled bool `json:"Disabled,optional"`
	Host     string `json:"Host,optional"`
	Port     int    `json:"Port,optional"`
	DB       int    `json:"DB,optional"`
	Crypto   bool   `json:"Crypto,optional"`
	// PasswordHex is AES-wrapped password (hex). Prefer env RECSYS_REDIS_PASSWORD_HEX when empty.
	PasswordHex    string `json:"PasswordHex,optional"`
	UserKeyPattern string `json:"UserKeyPattern,optional"`
	ItemKeyPattern string `json:"ItemKeyPattern,optional"`
}

// Config is the domain-agnostic rank HTTP service (scoring / model inference).
type Config struct {
	rest.RestConf
	RankEngine   RankEngineConfig            `json:"RankEngine,optional"`
	RankProfiles map[string]RankEngineConfig `json:"RankProfiles,optional"` // AB: rank_profile -> independent FM/TF bundle (online_map_rank multi-model).
	// RankModelBundles keys are "PreRankModelName|RankModelName" (see RankExpConf + ModelConf naming).
	RankModelBundles map[string]RankEngineConfig `json:"RankModelBundles,optional"`
	// RankExpConfPath loads online_map_rank RankExpConf.json (relative to this yaml directory if not absolute).
	RankExpConfPath string `json:"RankExpConfPath,optional"`
	FeatureRedis FeatureRedis                `json:"FeatureRedis,optional"`
}
