package sender

import "github.com/prometheus/client_golang/prometheus"

func registerMetrics(reg *prometheus.Registry, s *sender) {
	reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "push",
		Subsystem: "sender",
		Name:      "send_tokens",
		Help:      "total count of tokens",
	}, func() float64 {
		return float64(s.metrics.sendTokens.Load())
	}))
	reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "push",
		Subsystem: "sender",
		Name:      "send_count",
		Help:      "total count of send operations",
	}, func() float64 {
		return float64(s.metrics.sendCount.Load())
	}))
	s.metrics.sendDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "push",
		Subsystem: "snder",
		Name:      "duration_seconds",
		Objectives: map[float64]float64{
			0.5:  0.5,
			0.85: 0.01,
			0.95: 0.0005,
			0.99: 0.0001,
		},
	}, nil)
	reg.MustRegister(s.metrics.sendDuration)
}
