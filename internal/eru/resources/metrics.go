package resources

import (
	"github.com/projecteru2/core/utils"
	"github.com/projecteru2/yavirt/configs"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	lostGPUDesc = prometheus.NewDesc(
		prometheus.BuildFQName("vm", "gpu", "lost"),
		"Lost GPUs.",
		[]string{"node", "domain", "pci_addrs", "app_id", "app_sid", "appname", "ip"},
		nil)
	gpuPassthroughOKDesc = prometheus.NewDesc(
		prometheus.BuildFQName("node", "gpu_passthrough", "ok"),
		"GPU passthrough OK.",
		[]string{"node"},
		nil)
)

type MetricsCollector struct {
	mgr *Manager
}

func (mgr *Manager) GetMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		mgr: mgr,
	}
}

func (e *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- lostGPUDesc
	ch <- gpuPassthroughOKDesc
}

func (e *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	for _, v := range e.mgr.gpu.lostGPUCache.Items() {
		gpuInfo, _ := v.Object.(map[string]string)

		ch <- prometheus.MustNewConstMetric(
			lostGPUDesc,
			prometheus.GaugeValue,
			1.0,
			gpuInfo["node"],
			gpuInfo["domain"],
			gpuInfo["pci_addrs"],
			gpuInfo["app_id"],
			gpuInfo["app_sid"],
			gpuInfo["appname"],
			gpuInfo["ip"],
		)
	}
	ch <- prometheus.MustNewConstMetric(
		gpuPassthroughOKDesc,
		prometheus.GaugeValue,
		float64(utils.Bool2Int(e.mgr.gpu.passthroughOK.Load())),
		configs.Hostname(),
	)
}
