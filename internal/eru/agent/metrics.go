package agent

import (
	"context"
	"sync/atomic"

	"github.com/patrickmn/go-cache"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/core/utils"
	virttypes "github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/eru/types"
	"github.com/projecteru2/yavirt/internal/vmcache"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	vmHealthyDesc = prometheus.NewDesc(
		prometheus.BuildFQName("vm", "", "healthy"),
		"VM healthy status.",
		[]string{"EruID", "node", "app_id", "app_sid", "appname", "ip"},
		nil)
	coreHealthyDesc = prometheus.NewDesc(
		prometheus.BuildFQName("node", "core", "healthy"),
		"core healthy status.",
		[]string{"node"},
		nil)
)

type MetricsCollector struct {
	wrkStatusCache *cache.Cache
	coreHealthy    atomic.Bool
}

func (mgr *Manager) GetMetricsCollector() *MetricsCollector {
	return mgr.mCol
}

func (e *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- vmHealthyDesc
	ch <- coreHealthyDesc
}

func (e *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	logger := log.WithFunc("agent.MetricsCollector.Collect")
	for _, v := range e.wrkStatusCache.Items() {
		wrkStatus, _ := v.Object.(*types.WorkloadStatus)
		if wrkStatus == nil {
			logger.Warnf(context.TODO(), "[BUG] wrkStatus can't be nil here")
			continue
		}
		if !wrkStatus.Running {
			continue
		}
		de := vmcache.FetchDomainEntry(wrkStatus.ID)
		if de == nil {
			logger.Warnf(context.TODO(), "[eru agent] failed to get domain entry %s", wrkStatus.ID)
			continue
		}
		healthy := 0
		if wrkStatus.Healthy {
			healthy = 1
		}
		ch <- prometheus.MustNewConstMetric(
			vmHealthyDesc,
			prometheus.GaugeValue,
			float64(healthy),
			virttypes.EruID(wrkStatus.ID),
			wrkStatus.Nodename,
			de.AppID,
			de.AppSID,
			de.AppName,
			de.IP,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		coreHealthyDesc,
		prometheus.GaugeValue,
		float64(utils.Bool2Int(e.coreHealthy.Load())),
		configs.Hostname(),
	)
}
