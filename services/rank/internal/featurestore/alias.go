// Package featurestore re-exports pkg/featurestore for legacy imports.
package featurestore

import pkg "recsys_go/pkg/featurestore"

type (
	Fetcher        = pkg.Fetcher
	BatchFetcher   = pkg.BatchFetcher
	NoOpFetcher    = pkg.NoOpFetcher
	RedisJSONConfig = pkg.RedisJSONConfig
	RedisJSONFetcher = pkg.RedisJSONFetcher
	SparseKV       = pkg.SparseKV
)

var (
	NoOp                 = pkg.NoOp
	NewRedisJSONFetcher  = pkg.NewRedisJSONFetcher
	MergeUserItemJSON    = pkg.MergeUserItemJSON
)
