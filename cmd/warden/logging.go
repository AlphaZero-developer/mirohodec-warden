package main

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggingConfig struct {
	Level       string   `yaml:"level"`
	Development bool     `yaml:"development"`
	OutputPaths []string `yaml:"output_paths"`
	QueueSize   int      `yaml:"queue_size"`
	DropOnFull  *bool    `yaml:"drop_on_full"`
}

type AsyncLogger struct {
	base       *zap.Logger
	level      zap.AtomicLevel
	events     chan logEvent
	dropOnFull bool
	dropped    atomic.Uint64
	wg         sync.WaitGroup
}

type logEvent struct {
	level  zapcore.Level
	msg    string
	fields []zap.Field
}

func NewAsyncLogger(cfg LoggingConfig) (*AsyncLogger, error) {
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(normalizeLogLevel(cfg.Level))); err != nil {
		level.SetLevel(zapcore.InfoLevel)
	}

	zcfg := zap.NewProductionConfig()
	if cfg.Development {
		zcfg = zap.NewDevelopmentConfig()
	}
	zcfg.Level = level
	if len(cfg.OutputPaths) > 0 {
		zcfg.OutputPaths = cfg.OutputPaths
		zcfg.ErrorOutputPaths = cfg.OutputPaths
	}

	base, err := zcfg.Build()
	if err != nil {
		return nil, err
	}

	queueSize := cfg.QueueSize
	if queueSize <= 0 {
		queueSize = 2048
	}

	dropOnFull := true
	if cfg.DropOnFull != nil {
		dropOnFull = *cfg.DropOnFull
	}

	l := &AsyncLogger{
		base:       base,
		level:      level,
		events:     make(chan logEvent, queueSize),
		dropOnFull: dropOnFull,
	}
	l.wg.Add(1)
	go l.run()
	return l, nil
}

func (l *AsyncLogger) run() {
	defer l.wg.Done()
	for event := range l.events {
		switch event.level {
		case zapcore.DebugLevel:
			l.base.Debug(event.msg, event.fields...)
		case zapcore.WarnLevel:
			l.base.Warn(event.msg, event.fields...)
		case zapcore.ErrorLevel:
			l.base.Error(event.msg, event.fields...)
		default:
			l.base.Info(event.msg, event.fields...)
		}
	}
}

func (l *AsyncLogger) SetLevel(level string) {
	if l == nil {
		return
	}
	if err := l.level.UnmarshalText([]byte(normalizeLogLevel(level))); err != nil {
		return
	}
}

func (l *AsyncLogger) Debug(msg string, fields ...zap.Field) {
	l.enqueue(zapcore.DebugLevel, msg, fields...)
}

func (l *AsyncLogger) Info(msg string, fields ...zap.Field) {
	l.enqueue(zapcore.InfoLevel, msg, fields...)
}

func (l *AsyncLogger) Warn(msg string, fields ...zap.Field) {
	l.enqueue(zapcore.WarnLevel, msg, fields...)
}

func (l *AsyncLogger) Error(msg string, fields ...zap.Field) {
	l.enqueue(zapcore.ErrorLevel, msg, fields...)
}

func (l *AsyncLogger) enqueue(level zapcore.Level, msg string, fields ...zap.Field) {
	if l == nil {
		return
	}
	event := logEvent{level: level, msg: msg, fields: fields}
	if !l.dropOnFull {
		l.events <- event
		return
	}
	select {
	case l.events <- event:
	default:
		l.dropped.Add(1)
		select {
		case l.events <- logEvent{
			level:  zapcore.WarnLevel,
			msg:    "async logger queue is full",
			fields: []zap.Field{zap.Uint64("dropped", l.dropped.Load())},
		}:
		default:
		}
	}
}

func (l *AsyncLogger) Close() {
	if l == nil {
		return
	}
	close(l.events)
	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		_ = l.base.Sync()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
}

func normalizeLogLevel(level string) string {
	level = strings.TrimSpace(strings.ToLower(level))
	if level == "" {
		return "info"
	}
	return level
}
