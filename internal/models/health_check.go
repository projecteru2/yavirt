package models

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	erucluster "github.com/projecteru2/core/cluster"

	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func (g *Guest) healthCheckBridge() (*HealthCheckBridge, error) {
	raw, exists := g.JSONLabels[erucluster.LabelMeta]
	if !exists {
		return nil, errors.Annotatef(errors.ErrKeyNotExists, "no such label: %s", erucluster.LabelMeta)
	}

	hcb := &HealthCheckBridge{}
	return hcb, utils.JSONDecode([]byte(raw), hcb)
}

// HealthCheckBridge .
type HealthCheckBridge struct {
	Publish     []string
	HealthCheck struct {
		TCPPorts []string
		HTTPPort string
		HTTPURL  string
		HTTPCode int
	}
}

func (b *HealthCheckBridge) healthCheck(g *Guest) (hc HealthCheck, err error) {
	hc.IPNets = g.IPNets

	hc.TCPPorts = make([]int, len(b.HealthCheck.TCPPorts))
	for i, raw := range b.HealthCheck.TCPPorts {
		if hc.TCPPorts[i], err = strconv.Atoi(raw); err != nil {
			return
		}
	}

	if len(b.HealthCheck.HTTPPort) < 1 {
		b.HealthCheck.HTTPPort = "80"
	}
	if hc.HTTPPort, err = strconv.Atoi(b.HealthCheck.HTTPPort); err != nil {
		return
	}

	hc.HTTPPath = b.HealthCheck.HTTPURL
	hc.HTTPCode = b.HealthCheck.HTTPCode

	return
}

func (b *HealthCheckBridge) publishPorts() (ports []int, err error) {
	ports = make([]int, len(b.Publish))
	for i, raw := range b.Publish {
		if ports[i], err = strconv.Atoi(raw); err != nil {
			return
		}
	}
	return
}

// HealthCheck .
type HealthCheck struct {
	IPNets   meta.IPNets
	TCPPorts []int
	HTTPPort int
	HTTPPath string
	HTTPCode int
}

// HTTPEndpoints .
func (c HealthCheck) HTTPEndpoints() []string {
	if len(c.HTTPPath) < 1 {
		return nil
	}

	port := 80
	if c.HTTPPort > 0 {
		port = c.HTTPPort
	}

	path := strings.TrimLeft(c.HTTPPath, "/")

	endps := make([]string, len(c.IPNets))
	for i, ip := range c.IPNets {
		endps[i] = fmt.Sprintf("http://%s:%d/%s", ip.IPv4(), port, path) //nolint
	}

	return endps
}

// TCPEndpoints .
func (c HealthCheck) TCPEndpoints() []string {
	endps := make([]string, len(c.IPNets)*len(c.TCPPorts))

	for i, ip := range c.IPNets {
		for j, port := range c.TCPPorts {
			endps[i+j] = net.JoinHostPort(ip.IPv4(), strconv.Itoa(port))
		}
	}

	return endps
}
