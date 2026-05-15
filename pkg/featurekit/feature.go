// Package featurekit defines portable feature access patterns (Redis/Hive alignment is configured upstream).
package featurekit

// FieldID identifies a feature in training and serving (e.g. legacy FM field id).
type FieldID int32

// SparseEntry is a generic categorical feature with optional weight.
type SparseEntry struct {
	Field FieldID
	ID    int64
	W     float32
}
