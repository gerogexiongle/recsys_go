package recsyskit

// ExclusivePool holds ExclusiveRecallList results keyed by RecallType (C++ exclusive_item_info).
// These lanes are not merged into the main recall list before rank; show control (e.g. ForcedInsert) consumes them.
type ExclusivePool map[string][]ItemInfo
