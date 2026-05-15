package featurestore

import "fmt"

// Key patterns for Go lab / OSS (STRING JSON per entity).
//
// C++ online_map_center / online_map_rank use Redis HASH on pool UserFeature_0:
//   HGET {map_feature:proto:{id%10000}} FIELD {id}:outer
//   HGET {user_feature:proto:{uin%100000}} FIELD {uin}:game_exposure
// Other pools (OnlineData, MaterialData) hold invert ZSET / whitelist SET — not covered here yet.
//
// Go keeps one STRING per user/item for FM+filter sidecars; rank and center share the same fetcher.
const (
	DefaultUserKeyPattern = "recsysgo:user:%d"
	DefaultItemKeyPattern = "recsysgo:item:%d"
)

// KeyPatterns builds Redis STRING keys for FM / filter JSON documents.
type KeyPatterns struct {
	User string // fmt pattern with one %d (uin)
	Item string // fmt pattern with one %d (item/map id)
}

// DefaultKeyPatterns returns OSS lab prefixes (see scripts/seed_feature_redis.py).
func DefaultKeyPatterns() KeyPatterns {
	return KeyPatterns{User: DefaultUserKeyPattern, Item: DefaultItemKeyPattern}
}

func (p KeyPatterns) UserKey(uin int64) string {
	return fmt.Sprintf(p.User, uin)
}

func (p KeyPatterns) ItemKey(itemID int64) string {
	return fmt.Sprintf(p.Item, itemID)
}

// FutureKeyKinds documents C++-style keys for later adapters (invert / material).
type FutureKeyKinds struct {
	// InvertTagRecall: online_tag_recall_{SECTION}_5pcts_{tag_id} on OnlineData (ZSET).
	InvertTagRecall string
	// MaterialSuppress: cached suppress map list on MaterialData.
	MaterialSuppress string
}

// CppFutureKeys is reference naming only; not used by current Go fetcher.
func CppFutureKeys() FutureKeyKinds {
	return FutureKeyKinds{
		InvertTagRecall: "online_tag_recall_%s_5pcts_%d",
		MaterialSuppress: "online_suppress_map",
	}
}
