package metrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/internal/vmcache"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/digitalocean/go-libvirt"
)

var (
	libvirtUpDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "", "up"),
		"Whether scraping libvirt's metrics was successful.",
		[]string{"host"},
		nil)

	libvirtDomainNumbers = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "", "domains_number"),
		"Number of the domain",
		[]string{"host"},
		nil)

	libvirtDomainState = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "", "domain_state_code"),
		"Code of the domain state",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "stateDesc"},
		nil)

	libvirtDomainInfoNrVirtCPUDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_info", "virtual_cpus"),
		"Number of virtual CPUs for the domain.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)

	libvirtDomainInfoMaxMemDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_info", "maximum_memory_bytes"),
		"Maximum allowed memory of the domain, in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)
	libvirtDomainInfoMemoryDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_info", "memory_usage_bytes"),
		"Memory usage of the domain, in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)

	libvirtDomainInfoNrGPUDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_info", "gpus"),
		"Number of GPUs for the domain.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)

	// CPU stats
	libvirtDomainStatCPUTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_cpu_stats", "time_seconds_total"),
		"Amount of CPU time used by the domain, in seconds.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)

	// memory stats
	libvirtDomainStatMemorySwapInBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_mem_stats", "swap_in_bytes"),
		"Memory swap in of domain(the total amount of data read from swap space), in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)
	libvirtDomainStatMemorySwapOutBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_mem_stats", "swap_out_bytes"),
		"Memory swap out of the domain(the total amount of memory written out to swap space), in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)
	libvirtDomainStatMemoryUnusedBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_mem_stats", "unused_bytes"),
		"Memory unused of the domain, in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)
	libvirtDomainStatMemoryAvailableInBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_mem_stats", "available_bytes"),
		"Memory available of the domain, in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)
	libvirtDomainStatMemoryUsableBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_mem_stats", "usable_bytes"),
		"Memory usable of the domain(corresponds to 'Available' in /proc/meminfo), in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)
	libvirtDomainStatMemoryRssBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_mem_stats", "rss_bytes"),
		"Resident Set Size of the process running the domain, in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host"},
		nil)

	libvirtDomainBlockRdBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "read_bytes_total"),
		"Number of bytes read from a block device, in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "source_file", "target_device"},
		nil)
	libvirtDomainBlockRdReqDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "read_requests_total"),
		"Number of read requests from a block device.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "source_file", "target_device"},
		nil)
	libvirtDomainBlockWrBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "write_bytes_total"),
		"Number of bytes written from a block device, in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "source_file", "target_device"},
		nil)
	libvirtDomainBlockWrReqDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "write_requests_total"),
		"Number of write requests from a block device.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "source_file", "target_device"},
		nil)

	// DomainInterface
	libvirtDomainInterfaceRxBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "rx_bytes_total"),
		"Number of bytes received on a network interface, in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "target_device"},
		nil)
	libvirtDomainInterfaceRxPacketsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "rx_packets_total"),
		"Number of packets received on a network interface.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "target_device"},
		nil)
	libvirtDomainInterfaceRxErrsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "rx_errors_total"),
		"Number of packet receive errors on a network interface.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "target_device"},
		nil)
	libvirtDomainInterfaceRxDropDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "rx_drops_total"),
		"Number of packet receive drops on a network interface.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "target_device"},
		nil)
	libvirtDomainInterfaceTxBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "tx_bytes_total"),
		"Number of bytes transmitted on a network interface, in bytes.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "target_device"},
		nil)
	libvirtDomainInterfaceTxPacketsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "tx_packets_total"),
		"Number of packets transmitted on a network interface.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "target_device"},
		nil)
	libvirtDomainInterfaceTxErrsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "tx_errors_total"),
		"Number of packet transmit errors on a network interface.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "target_device"},
		nil)
	libvirtDomainInterfaceTxDropDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "tx_drops_total"),
		"Number of packet transmit drops on a network interface.",
		[]string{"name", "ip", "eruName", "instanceId", "userName", "userId", "host", "target_device"},
		nil)

	domainState = map[libvirt.DomainState]string{
		libvirt.DomainNostate:     "no state",
		libvirt.DomainRunning:     "the domain is running",
		libvirt.DomainBlocked:     "the domain is blocked on resource",
		libvirt.DomainPaused:      "the domain is paused by user",
		libvirt.DomainShutdown:    "the domain is being shut down",
		libvirt.DomainShutoff:     "the domain is shut off",
		libvirt.DomainCrashed:     "the domain is crashed",
		libvirt.DomainPmsuspended: "the domain is suspended by guest power management",
		// libvirtgo.DOMAIN_LAST:        "this enum value will increase over time as new events are added to the libvirt API",
	}
)

type collectFunc func(ch chan<- prometheus.Metric, stats *vmcache.DomainStatsResp, promLabels []string) (err error)

// LibvirtExporter implements a Prometheus exporter for libvirt state.
type LibvirtExporter struct {
	host string
}

// NewLibvirtExporter creates a new Prometheus exporter for libvirt.
func NewLibvirtExporter(hn string) *LibvirtExporter {
	return &LibvirtExporter{hn}
}

// Collect scrapes Prometheus metrics from libvirt.
func (e *LibvirtExporter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		libvirtUpDesc,
		prometheus.GaugeValue,
		1.0,
		e.host)

	domainMap := vmcache.FetchStats()
	domainNumber := len(domainMap)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainNumbers,
		prometheus.GaugeValue,
		float64(domainNumber),
		e.host,
	)

	for name := range domainMap {
		dom := domainMap[name]
		if err := e.CollectDomain(ch, &dom); err != nil {
			log.WithFunc("metrics.Collect").Errorf(context.TODO(), err, "failed to collect domain %s", name)
			return
		}
	}
}

// CollectDomain extracts Prometheus metrics from a libvirt domain.
func (e *LibvirtExporter) CollectDomain(ch chan<- prometheus.Metric, stats *vmcache.DomainStatsResp) (err error) {
	// if stats.State == nil || stats.Balloon == nil || stats.Cpu == nil {
	// 	return nil
	// }
	var (
		rState      = stats.State
		rvirCPU, _  = vmcache.ToUint64(stats.Stats["vcpu.maximum"])
		nrGPUs      = len(stats.GPUAddrs)
		rmaxmem, _  = vmcache.ToUint64(stats.Stats["balloon.maximum"])
		rmemory, _  = vmcache.ToUint64(stats.Stats["balloon.current"])
		rcputime, _ = vmcache.ToUint64(stats.Stats["cpu.time"])
	)

	promLabels := []string{
		stats.AppName,
		stats.IP,
		stats.EruName,
		stats.UUID,
		stats.UserName,
		stats.UserID,
		e.host}

	ch <- prometheus.MustNewConstMetric(libvirtDomainState, prometheus.GaugeValue, float64(rState), append(promLabels, domainState[rState])...)

	ch <- prometheus.MustNewConstMetric(libvirtDomainInfoMaxMemDesc, prometheus.GaugeValue, float64(rmaxmem)*1024, promLabels...)
	ch <- prometheus.MustNewConstMetric(libvirtDomainInfoMemoryDesc, prometheus.GaugeValue, float64(rmemory)*1024, promLabels...)
	ch <- prometheus.MustNewConstMetric(libvirtDomainInfoNrVirtCPUDesc, prometheus.GaugeValue, float64(rvirCPU), promLabels...)
	ch <- prometheus.MustNewConstMetric(libvirtDomainStatCPUTimeDesc, prometheus.CounterValue, float64(rcputime)/1e9, promLabels...)
	ch <- prometheus.MustNewConstMetric(libvirtDomainInfoNrGPUDesc, prometheus.GaugeValue, float64(nrGPUs), promLabels...)
	// var isActive int32
	// if isActive, err = l.DomainIsActive(domain.libvirtDomain); err != nil {
	// 	logger.Error("failed to get active status of domain", zap.String("name", "ip", domain.domainName), zap.Error(err))
	// 	return err
	// }
	// if isActive != 1 {
	// 	logger.Info("domain is not active", zap.String("name", "ip", domain.domainName))
	// 	return nil
	// }

	for _, collectFunc := range []collectFunc{CollectDomainBlockDeviceInfo, CollectDomainNetworkInfo, CollectDomainMemoryStatInfo} {
		if err = collectFunc(ch, stats, promLabels); err != nil {
			log.WithFunc("metrics.CollectDomain").Errorf(context.TODO(), err, "failed to collect some domain info")
		}
	}

	return nil
}

func CollectDomainBlockDeviceInfo(ch chan<- prometheus.Metric, stats *vmcache.DomainStatsResp, promLabels []string) (err error) {
	// Report block device statistics.
	count, _ := vmcache.ToUint64(stats.Stats["block.count"])

	for i := uint64(0); i < count; i++ {
		diskName, _ := vmcache.ToString(stats.Stats[fmt.Sprintf("block.%d.name", i)])
		if !strings.HasPrefix(diskName, "vd") {
			continue
		}
		diskPath, _ := vmcache.ToString(stats.Stats[fmt.Sprintf("block.%d.path", i)])
		rRdReq, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("block.%d.rd.reqs", i)])
		rRdBytes, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("block.%d.rd.bytes", i)])
		rWrReq, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("block.%d.wr.reqs", i)])
		rWrBytes, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("block.%d.wr.bytes", i)])

		promDiskLabels := append(promLabels, diskPath, diskName) //nolint
		ch <- prometheus.MustNewConstMetric(
			libvirtDomainBlockRdBytesDesc,
			prometheus.CounterValue,
			float64(rRdBytes),
			promDiskLabels...)

		ch <- prometheus.MustNewConstMetric(
			libvirtDomainBlockRdReqDesc,
			prometheus.CounterValue,
			float64(rRdReq),
			promDiskLabels...)

		ch <- prometheus.MustNewConstMetric(
			libvirtDomainBlockWrBytesDesc,
			prometheus.CounterValue,
			float64(rWrBytes),
			promDiskLabels...)

		ch <- prometheus.MustNewConstMetric(
			libvirtDomainBlockWrReqDesc,
			prometheus.CounterValue,
			float64(rWrReq),
			promDiskLabels...)

	}
	return
}

func CollectDomainNetworkInfo(ch chan<- prometheus.Metric, stats *vmcache.DomainStatsResp, promLabels []string) (err error) {
	// Report network interface statistics.
	count, _ := vmcache.ToUint64(stats.Stats["net.count"])
	for idx := uint64(0); idx < count; idx++ {
		ifaceName, _ := vmcache.ToString(stats.Stats[fmt.Sprintf("net.%d.name", idx)])
		if ifaceName == "" {
			continue
		}
		rRxBytes, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("net.%d.rx.bytes", idx)])
		rRxPackets, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("net.%d.rx.pkts", idx)])
		rRxErrs, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("net.%d.rx.errs", idx)])
		rRxDrop, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("net.%d.rx.drop", idx)])
		rTxBytes, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("net.%d.tx.bytes", idx)])
		rTxPackets, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("net.%d.tx.pkts", idx)])
		rTxErrs, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("net.%d.tx.errs", idx)])
		rTxDrop, _ := vmcache.ToUint64(stats.Stats[fmt.Sprintf("net.%d.tx.drop", idx)])

		promInterfaceLabels := append(promLabels, ifaceName) //nolint
		ch <- prometheus.MustNewConstMetric(
			libvirtDomainInterfaceRxBytesDesc,
			prometheus.CounterValue,
			float64(rRxBytes),
			promInterfaceLabels...)

		ch <- prometheus.MustNewConstMetric(
			libvirtDomainInterfaceRxPacketsDesc,
			prometheus.CounterValue,
			float64(rRxPackets),
			promInterfaceLabels...)

		ch <- prometheus.MustNewConstMetric(
			libvirtDomainInterfaceRxErrsDesc,
			prometheus.CounterValue,
			float64(rRxErrs),
			promInterfaceLabels...)

		ch <- prometheus.MustNewConstMetric(
			libvirtDomainInterfaceRxDropDesc,
			prometheus.CounterValue,
			float64(rRxDrop),
			promInterfaceLabels...)

		ch <- prometheus.MustNewConstMetric(
			libvirtDomainInterfaceTxBytesDesc,
			prometheus.CounterValue,
			float64(rTxBytes),
			promInterfaceLabels...)

		ch <- prometheus.MustNewConstMetric(
			libvirtDomainInterfaceTxPacketsDesc,
			prometheus.CounterValue,
			float64(rTxPackets),
			promInterfaceLabels...)

		ch <- prometheus.MustNewConstMetric(
			libvirtDomainInterfaceTxErrsDesc,
			prometheus.CounterValue,
			float64(rTxErrs),
			promInterfaceLabels...)

		ch <- prometheus.MustNewConstMetric(
			libvirtDomainInterfaceTxDropDesc,
			prometheus.CounterValue,
			float64(rTxDrop),
			promInterfaceLabels...)
	}
	return err
}

func CollectDomainMemoryStatInfo(ch chan<- prometheus.Metric, stats *vmcache.DomainStatsResp, promLabels []string) (err error) {
	var (
		swapIn, _    = vmcache.ToUint64(stats.Stats["balloon.swap_in"])
		swapOut, _   = vmcache.ToUint64(stats.Stats["balloon.swap_out"])
		unused, _    = vmcache.ToUint64(stats.Stats["balloon.unused"])
		available, _ = vmcache.ToUint64(stats.Stats["balloon.available"])
		usable, _    = vmcache.ToUint64(stats.Stats["balloon.usable"])
		rss, _       = vmcache.ToUint64(stats.Stats["balloon.rss"])
	)

	ch <- prometheus.MustNewConstMetric(
		libvirtDomainStatMemorySwapInBytesDesc,
		prometheus.GaugeValue,
		float64(swapIn)*1024,
		promLabels...)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainStatMemorySwapOutBytesDesc,
		prometheus.GaugeValue,
		float64(swapOut)*1024,
		promLabels...)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainStatMemoryUnusedBytesDesc,
		prometheus.GaugeValue,
		float64(unused*1024),
		promLabels...)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainStatMemoryAvailableInBytesDesc,
		prometheus.GaugeValue,
		float64(available*1024),
		promLabels...)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainStatMemoryUsableBytesDesc,
		prometheus.GaugeValue,
		float64(usable*1024),
		promLabels...)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainStatMemoryRssBytesDesc,
		prometheus.GaugeValue,
		float64(rss*1024),
		promLabels...)
	return
}

// Describe returns metadata for all Prometheus metrics that may be exported.
func (e *LibvirtExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- libvirtUpDesc
	ch <- libvirtDomainNumbers

	// domain info
	ch <- libvirtDomainState
	ch <- libvirtDomainInfoMaxMemDesc
	ch <- libvirtDomainInfoMemoryDesc
	ch <- libvirtDomainInfoNrVirtCPUDesc
	ch <- libvirtDomainStatCPUTimeDesc
	ch <- libvirtDomainInfoNrGPUDesc

	// domain block
	ch <- libvirtDomainBlockRdBytesDesc
	ch <- libvirtDomainBlockRdReqDesc
	ch <- libvirtDomainBlockWrBytesDesc
	ch <- libvirtDomainBlockWrReqDesc

	// domain interface
	ch <- libvirtDomainInterfaceRxBytesDesc
	ch <- libvirtDomainInterfaceRxPacketsDesc
	ch <- libvirtDomainInterfaceRxErrsDesc
	ch <- libvirtDomainInterfaceRxDropDesc
	ch <- libvirtDomainInterfaceTxBytesDesc
	ch <- libvirtDomainInterfaceTxPacketsDesc
	ch <- libvirtDomainInterfaceTxErrsDesc
	ch <- libvirtDomainInterfaceTxDropDesc

	// domain mem stat
	ch <- libvirtDomainStatMemorySwapInBytesDesc
	ch <- libvirtDomainStatMemorySwapOutBytesDesc
	ch <- libvirtDomainStatMemoryUnusedBytesDesc
	ch <- libvirtDomainStatMemoryAvailableInBytesDesc
	ch <- libvirtDomainStatMemoryUsableBytesDesc
	ch <- libvirtDomainStatMemoryRssBytesDesc
}
