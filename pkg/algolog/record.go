package algolog

import (
	"fmt"
	"strconv"
	"strings"
)

// Record mirrors C++ KafkaClient/AlgorithmData.h pipe-separated wire format.
type Record struct {
	ServerName     string
	Timestamp      int64
	APIType        int
	DataType       string
	Channel        string
	AppVersion     string
	OSType         string
	Resolution     string
	TerminalModel  string
	DeviceID       string
	Section        string
	RentSection    string
	Country        string
	LangList       string
	IPv4           string
	IPv6           string
	DisAlgo        int
	Standby1       string
	Standby2       string
	RequestID      string
	UIN            int64
	Standby3       string
	ExpList        string
	Standby4       string
	MaterialList   string
	Standby5       string
	Standby6       string
	Standby7       string
}

// Serialize returns the C++ AlgorithmData::Serialize() pipe string.
func (r Record) Serialize() string {
	parts := []string{
		r.ServerName,
		strconv.FormatInt(r.Timestamp, 10),
		strconv.Itoa(r.APIType),
		r.DataType,
		r.Channel,
		r.AppVersion,
		r.OSType,
		r.Resolution,
		r.TerminalModel,
		r.DeviceID,
		r.Section,
		r.RentSection,
		r.Country,
		r.LangList,
		r.IPv4,
		r.IPv6,
		strconv.Itoa(r.DisAlgo),
		r.Standby1,
		r.Standby2,
		r.RequestID,
		strconv.FormatInt(r.UIN, 10),
		r.Standby3,
		r.ExpList,
		r.Standby4,
		r.MaterialList,
		r.Standby5,
		r.Standby6,
		r.Standby7,
	}
	return strings.Join(parts, "|")
}

// MaterialEntry is one item in material_list (C++ main_item_list log slot).
type MaterialEntry struct {
	ItemID     int64
	RecallType string
	RankScore  float64
	ShowScore  float64
	TaskID     int
	PreScore   float64
	ReScore    float64
}

// FormatMaterialList builds [id:recall:rank:show:task:pre:re,...] (max 18 items, same as C++).
func FormatMaterialList(entries []MaterialEntry, max int) string {
	if max <= 0 {
		max = 18
	}
	if len(entries) > max {
		entries = entries[:max]
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, e := range entries {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d:%s:%s:%s:%d:%s:%s",
			e.ItemID,
			e.RecallType,
			scoreStr(e.RankScore),
			scoreStr(e.ShowScore),
			e.TaskID,
			scoreStr(e.PreScore),
			scoreStr(e.ReScore),
		)
	}
	b.WriteByte(']')
	return b.String()
}

func scoreStr(v float64) string {
	return strconv.FormatFloat(v, 'f', 7, 64)
}
