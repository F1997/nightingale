package cstats

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// 服务名称 n9e-center
const Service = "n9e-center"

var (
	labels = []string{"service", "code", "path", "method"}

	// 用于记录HTTP服务的运行时间
	uptime = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "uptime",
			Help: "HTTP service uptime.",
		}, []string{"service"},
	)
	// 用于记录HTTP请求的总数
	RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_count_total",
			Help: "Total number of HTTP requests made.",
		}, labels,
	)

	// 记录HTTP请求的响应时间
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Buckets: []float64{.01, .1, 1, 10},
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latencies in seconds.",
		}, labels,
	)
)

// 初始化Prometheus指标，并将它们注册到Prometheus的默认注册表中
func Init() {
	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(
		uptime,
		RequestCounter,
		RequestDuration,
	)

	go recordUptime()
}

// recordUptime increases service uptime per second.
func recordUptime() {
	for range time.Tick(time.Second) {
		uptime.WithLabelValues(Service).Inc()
	}
}
