package memsto

import "github.com/prometheus/client_golang/prometheus"

// 储各种 Prometheus 指标
type Stats struct {
	GaugeCronDuration *prometheus.GaugeVec // 记录 Cron 方法的执行时间
	GaugeSyncNumber   *prometheus.GaugeVec // 记录 Cron 同步的数量
}

func NewSyncStats() *Stats {
	GaugeCronDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "n9e",
		Subsystem: "cron",
		Name:      "duration",
		Help:      "Cron method use duration, unit: ms.",
	}, []string{"name"})

	GaugeSyncNumber := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "n9e",
		Subsystem: "cron",
		Name:      "sync_number",
		Help:      "Cron sync number.",
	}, []string{"name"})

	prometheus.MustRegister(
		GaugeCronDuration,
		GaugeSyncNumber,
	)

	return &Stats{
		GaugeCronDuration: GaugeCronDuration,
		GaugeSyncNumber:   GaugeSyncNumber,
	}
}
