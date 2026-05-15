package centerconfig

import (
	"encoding/json"
	"os"
)

// LoadRecallLibrary loads Config_Recall-shaped JSON.
func LoadRecallLibrary(path string) (*RecallLibrary, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lib RecallLibrary
	if err := json.Unmarshal(b, &lib); err != nil {
		return nil, err
	}
	return &lib, nil
}

// LoadFilterLibrary loads Config_Filter.MapRecommend-shaped JSON.
func LoadFilterLibrary(path string) (*FilterLibrary, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lib FilterLibrary
	if err := json.Unmarshal(b, &lib); err != nil {
		return nil, err
	}
	return &lib, nil
}

// LoadShowLibrary loads Config_ShowControl.MapRecommend-shaped JSON.
func LoadShowLibrary(path string) (*ShowLibrary, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lib ShowLibrary
	if err := json.Unmarshal(b, &lib); err != nil {
		return nil, err
	}
	return &lib, nil
}
