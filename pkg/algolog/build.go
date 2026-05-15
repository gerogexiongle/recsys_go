package algolog

import (
	"encoding/json"
	"os"
	"strconv"
	"time"

	"recsys_go/pkg/recsyskit"
)

// Default wire ids for OSS demo (avoid production map-specific api_type / data_type).
const DefaultAPIType = 10001
const DefaultDataType = "cn_ol_item"

// Input carries request context and final ranked items for algorithm logging.
type Input struct {
	UUID            string
	UserID          int64
	Section         int32
	ExpIDs          []int32
	DisablePersonal int32
	DeviceID        string
	TerminalModel   string
	OSType          string
	Items           []recsyskit.ItemInfo
	APIType         int
	DataType        string
}

// BuildRecord creates a C++-compatible AlgorithmData record.
func BuildRecord(in Input) Record {
	host, _ := os.Hostname()
	if in.APIType == 0 {
		in.APIType = DefaultAPIType
	}
	if in.DataType == "" {
		in.DataType = DefaultDataType
	}
	entries := make([]MaterialEntry, 0, len(in.Items))
	for _, it := range in.Items {
		taskID := 0
		if it.Extra != nil {
			if s := it.Extra["task_id"]; s != "" {
				taskID, _ = strconv.Atoi(s)
			}
		}
		sc := it.Score
		entries = append(entries, MaterialEntry{
			ItemID:     int64(it.ID),
			RecallType: it.RecallType,
			RankScore:  sc,
			ShowScore:  sc,
			TaskID:     taskID,
		})
	}
	return Record{
		ServerName:    host,
		Timestamp:     time.Now().Unix(),
		APIType:       in.APIType,
		DataType:      in.DataType,
		OSType:        in.OSType,
		TerminalModel: in.TerminalModel,
		DeviceID:      in.DeviceID,
		Section:       strconv.FormatInt(int64(in.Section), 10),
		DisAlgo:       int(in.DisablePersonal),
		RequestID:     in.UUID,
		UIN:           in.UserID,
		ExpList:       formatExpList(in.ExpIDs),
		MaterialList:  FormatMaterialList(entries, 18),
	}
}

// JSONSnapshot adds a Go-friendly JSON view for local debug (not sent to Kafka by default).
func JSONSnapshot(in Input, rec Record) []byte {
	type item struct {
		ItemID     int64   `json:"item_id"`
		RecallType string  `json:"recall_type"`
		Score      float64 `json:"score"`
	}
	items := make([]item, 0, len(in.Items))
	for _, it := range in.Items {
		items = append(items, item{ItemID: int64(it.ID), RecallType: it.RecallType, Score: it.Score})
	}
	out := map[string]any{
		"request_id":  in.UUID,
		"uin":         in.UserID,
		"exp_ids":     in.ExpIDs,
		"section":     in.Section,
		"dis_algo":    in.DisablePersonal,
		"api_type":    rec.APIType,
		"data_type":   rec.DataType,
		"items":       items,
		"wire_format": rec.Serialize(),
	}
	b, _ := json.Marshal(out)
	return b
}

func formatExpList(exp []int32) string {
	if len(exp) == 0 {
		return "[]"
	}
	b, err := json.Marshal(exp)
	if err != nil {
		return "[]"
	}
	return string(b)
}
