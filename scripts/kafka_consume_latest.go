// One-shot: read latest messages from topic (local E2E). Usage:
//   go run scripts/kafka_consume_latest.go -brokers 127.0.0.1:9092 -topic test -timeout 8s
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

func main() {
	brokers := flag.String("brokers", "127.0.0.1:9092", "comma-separated brokers")
	topic := flag.String("topic", "test", "topic")
	match := flag.String("match", "", "if set, return last message containing this substring")
	timeout := flag.Duration("timeout", 8*time.Second, "read timeout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     strings.Split(*brokers, ","),
		Topic:       *topic,
		GroupID:     fmt.Sprintf("recsys-e2e-%d", time.Now().UnixNano()),
		StartOffset: kafka.FirstOffset,
	})
	defer r.Close()

	deadline := time.Now().Add(*timeout)
	var last string
	for time.Now().Before(deadline) {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			break
		}
		s := string(m.Value)
		last = s
		if *match != "" && strings.Contains(s, *match) {
			last = s
			break
		}
	}
	if last == "" {
		fmt.Fprintln(os.Stderr, "no kafka message received")
		os.Exit(1)
	}
	fmt.Print(last)
}
