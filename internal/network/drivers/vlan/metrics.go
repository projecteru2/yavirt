package vlan

import (
	"sync/atomic"

	"github.com/projecteru2/core/utils"
	"github.com/projecteru2/yavirt/configs"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	vlanHealthyDesc = prometheus.NewDesc(
		prometheus.BuildFQName("network", "vlan", "healthy"),
		"vlan healthy status.",
		[]string{"node"},
		nil)
)

type MetricsCollector struct {
	healthy atomic.Bool
}

func (d *Handler) GetMetricsCollector() prometheus.Collector {
	return d.mCol
}

func (e *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- vlanHealthyDesc
}

func (e *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	healthy := utils.Bool2Int(e.healthy.Load())
	ch <- prometheus.MustNewConstMetric(
		vlanHealthyDesc,
		prometheus.GaugeValue,
		float64(healthy),
		configs.Hostname(),
	)
}
