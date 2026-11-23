package main

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"`
	Path    string `yaml:"path"`
}

type Metrics struct {
	registry              *prometheus.Registry
	updatesTotal          prometheus.Counter
	messagesTotal         prometheus.Counter
	deletedJailedMessages prometheus.Counter
	deletedSpamMessages   prometheus.Counter
	moderationActions     *prometheus.CounterVec
	reloadsTotal          *prometheus.CounterVec
}

func NewMetrics(store RuntimeStore) *Metrics {
	registry := prometheus.NewRegistry()
	m := &Metrics{
		registry: registry,
		updatesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "warden_updates_total",
			Help: "Total Telegram updates processed by the bot.",
		}),
		messagesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "warden_messages_total",
			Help: "Total Telegram messages seen by the bot.",
		}),
		deletedJailedMessages: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "warden_jailed_messages_deleted_total",
			Help: "Messages deleted because jailed users wrote outside the rehab topic.",
		}),
		deletedSpamMessages: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "warden_spam_messages_deleted_total",
			Help: "Messages deleted by the spam filter.",
		}),
		moderationActions: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "warden_moderation_actions_total",
			Help: "Moderation actions executed by command.",
		}, []string{"action"}),
		reloadsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "warden_reloads_total",
			Help: "Hot reload attempts by target and result.",
		}, []string{"target", "result"}),
	}

	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry.MustRegister(
		m.updatesTotal,
		m.messagesTotal,
		m.deletedJailedMessages,
		m.deletedSpamMessages,
		m.moderationActions,
		m.reloadsTotal,
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "warden_known_users",
			Help: "Known non-bot users in storage.",
		}, func() float64 {
			return float64(store.Counts(time.Now().UTC()).KnownUsers)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "warden_active_rehab_users",
			Help: "Users currently in rehab room.",
		}, func() float64 {
			return float64(store.Counts(time.Now().UTC()).ActiveJailed)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "warden_active_mutes",
			Help: "Users currently muted by the bot.",
		}, func() float64 {
			return float64(store.Counts(time.Now().UTC()).ActiveMuted)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "warden_forever_bans",
			Help: "Users stored in the forever-ban list.",
		}, func() float64 {
			return float64(store.Counts(time.Now().UTC()).ForeverBans)
		}),
	)

	return m
}

func (m *Metrics) Start(ctx context.Context, cfg MetricsConfig, log *AsyncLogger) {
	if m == nil || !cfg.Enabled {
		return
	}
	listen := cfg.Listen
	if listen == "" {
		listen = ":2112"
	}
	path := cfg.Path
	if path == "" {
		path = "/metrics"
	}

	mux := http.NewServeMux()
	mux.Handle(path, promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Warn("metrics server shutdown failed", zap.Error(err))
		}
	}()

	log.Info("metrics server started", zap.String("listen", listen), zap.String("path", path))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("metrics server stopped with error", zap.Error(err))
	}
}

func (m *Metrics) IncUpdate() {
	if m != nil {
		m.updatesTotal.Inc()
	}
}

func (m *Metrics) IncMessage() {
	if m != nil {
		m.messagesTotal.Inc()
	}
}

func (m *Metrics) IncDeletedJailedMessage() {
	if m != nil {
		m.deletedJailedMessages.Inc()
	}
}

func (m *Metrics) IncDeletedSpamMessage() {
	if m != nil {
		m.deletedSpamMessages.Inc()
	}
}

func (m *Metrics) IncModeration(action string) {
	if m != nil {
		m.moderationActions.WithLabelValues(action).Inc()
	}
}

func (m *Metrics) IncReload(target, result string) {
	if m != nil {
		m.reloadsTotal.WithLabelValues(target, result).Inc()
	}
}
