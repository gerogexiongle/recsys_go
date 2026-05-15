package recall

import (
	"encoding/json"
	"strings"
)

const (
	UserGroupDefault  = "def_group"
	UserGroupNewUser  = "T0_NewUser" // C++ Config_Recall new-user bucket
)

// ResolveUserGroup picks recall/filter/show UserGroup (request override, else user feat).
func ResolveUserGroup(reqGroup string, userFeatJSON []byte) string {
	if g := strings.TrimSpace(reqGroup); g != "" {
		return g
	}
	var doc struct {
		UserSegment string `json:"user_segment"`
		IsNewUser   bool   `json:"is_new_user"`
	}
	if len(userFeatJSON) > 0 {
		_ = json.Unmarshal(userFeatJSON, &doc)
	}
	if doc.UserSegment == UserGroupNewUser || doc.IsNewUser {
		return UserGroupNewUser
	}
	return UserGroupDefault
}
