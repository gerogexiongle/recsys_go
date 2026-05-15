package recsyskit

// RequestContext is transport-agnostic context for one recommendation request.
type RequestContext struct {
	UUID            string
	UserID          int64
	Section         int32
	ExpIDs          []int32
	DisablePersonal int32
	DeviceID        string
	TerminalModel   string
	OSType          string
	// UserGroup selects MapRecommend.UserGroupList entry (e.g. def_group / T0_NewUser).
	UserGroup string
	// Exposure counts impressions per item for filter policies (filled by center / adapters).
	Exposure map[ItemID]int
}
