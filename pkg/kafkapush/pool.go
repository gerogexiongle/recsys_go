package kafkapush

import (
	"context"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// Pool async-pushes algorithm log strings (C++ PushKafkaPool).
type Pool struct {
	cfg    Config
	ch     chan string
	writer *kafka.Writer
	wg     sync.WaitGroup
	once   sync.Once
}

// Enabled reports whether the pool will accept messages.
func (p *Pool) Enabled() bool {
	return p != nil && p.cfg.EnabledOn()
}

func (p *Pool) APIType() int {
	if p.cfg.APIType != 0 {
		return p.cfg.APIType
	}
	return 0
}

func (p *Pool) DataType() string {
	if p.cfg.DataType != "" {
		return p.cfg.DataType
	}
	return ""
}

// New creates a push pool; disabled config returns a no-op pool.
func New(cfg Config) (*Pool, error) {
	p := &Pool{cfg: cfg}
	if !cfg.EnabledOn() {
		return p, nil
	}
	p.ch = make(chan string, cfg.queueSize())
	p.writer = &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Topic:                  cfg.Topic,
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireOne,
		Async:                  true,
		BatchTimeout:           200 * time.Millisecond,
		WriteTimeout:           5 * time.Second,
		AllowAutoTopicCreation: true,
	}
	p.wg.Add(1)
	go p.loop()
	return p, nil
}

func (p *Pool) loop() {
	defer p.wg.Done()
	for msg := range p.ch {
		if p.writer == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = p.writer.WriteMessages(ctx, kafka.Message{Value: []byte(msg)})
		cancel()
	}
}

// Push enqueues a serialized algorithm log (non-blocking; drops when queue full).
func (p *Pool) Push(msg string) {
	if !p.Enabled() || p.ch == nil || msg == "" {
		return
	}
	select {
	case p.ch <- msg:
	default:
	}
}

// Close drains and shuts down the producer.
func (p *Pool) Close() error {
	p.once.Do(func() {
		if p.ch != nil {
			close(p.ch)
		}
		p.wg.Wait()
		if p.writer != nil {
			_ = p.writer.Close()
		}
	})
	return nil
}
