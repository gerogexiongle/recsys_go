package centerconfig

// CenterBundle groups split center JSON configs (mirrors C++ Config_Recall + Config_Filter + Config_ShowControl).
// Recall must be non-nil for the recommend service to run the center pipeline.
type CenterBundle struct {
	Recall *RecallLibrary
	Filter *FilterLibrary
	Show   *ShowLibrary
}
