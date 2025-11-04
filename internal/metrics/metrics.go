package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	QueueSize    = prometheus.NewGauge(prometheus.GaugeOpts{Name: "mm_queue_size", Help: "current queue size"})
	MatchesTotal = prometheus.NewCounter(prometheus.CounterOpts{Name: "mm_matches_total", Help: "total matches formed"})
)

func Init() {
	prometheus.MustRegister(QueueSize, MatchesTotal)
}
