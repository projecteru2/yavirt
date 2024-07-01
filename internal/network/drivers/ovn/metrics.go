package ovn

import (
	"sync/atomic"

	"github.com/projecteru2/core/utils"
	"github.com/projecteru2/yavirt/configs"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	calicoHealthyDesc = prometheus.NewDesc(
		prometheus.BuildFQName("node", "ovn", "healthy"),
		"ovn healthy status.",
		[]string{"node"},
		nil)
)

type MetricsCollector struct {
	healthy atomic.Bool
}

func (d *Driver) GetMetricsCollector() prometheus.Collector {
	return d.mCol
}

func (e *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- calicoHealthyDesc
}

func (e *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	healthy := utils.Bool2Int(e.healthy.Load())
	ch <- prometheus.MustNewConstMetric(
		calicoHealthyDesc,
		prometheus.GaugeValue,
		float64(healthy),
		configs.Hostname(),
	)
}
