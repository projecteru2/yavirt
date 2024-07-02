package boar

import (
	"sync/atomic"

	"github.com/projecteru2/core/utils"
	"github.com/projecteru2/yavirt/configs"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	imageHubHealthyDesc = prometheus.NewDesc(
		prometheus.BuildFQName("node", "image_hub", "healthy"),
		"image hub healthy status.",
		[]string{"node"},
		nil)
	libvirtHealthyDesc = prometheus.NewDesc(
		prometheus.BuildFQName("node", "libvirt", "healthy"),
		"libvirt healthy status.",
		[]string{"node"},
		nil)
)

type MetricsCollector struct {
	imageHealthy   atomic.Bool
	libvirtHealthy atomic.Bool
}

func (d *Boar) GetMetricsCollector() prometheus.Collector {
	return d.mCol
}

func (e *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- imageHubHealthyDesc
	ch <- libvirtHealthyDesc
}

func (e *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		imageHubHealthyDesc,
		prometheus.GaugeValue,
		float64(utils.Bool2Int(e.imageHealthy.Load())),
		configs.Hostname(),
	)
	ch <- prometheus.MustNewConstMetric(
		libvirtHealthyDesc,
		prometheus.GaugeValue,
		float64(utils.Bool2Int(e.libvirtHealthy.Load())),
		configs.Hostname(),
	)
}
