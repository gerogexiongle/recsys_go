package kafkapush

// Config mirrors C++ AppConf KafkaPush + TDKafka (subset for OSS/local).
type Config struct {
	Enabled   bool     `json:"Enabled,optional"`
	Brokers   []string `json:"Brokers,optional"`
	Topic     string   `json:"Topic,optional"`
	QueueSize int      `json:"QueueSize,optional"`
	// Algorithm log wire defaults (override per env; OSS uses cn_ol_item / 10001).
	DataType string `json:"DataType,optional"`
	APIType  int    `json:"APIType,optional"`
}

func (c Config) queueSize() int {
	if c.QueueSize <= 0 {
		return 2048
	}
	return c.QueueSize
}

// Enabled reports whether Kafka push is on.
func (c Config) EnabledOn() bool {
	return c.Enabled && len(c.Brokers) > 0 && c.Topic != ""
}
