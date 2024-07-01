package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/projecteru2/yavirt/internal/service"
	"github.com/projecteru2/yavirt/internal/utils"

	"github.com/projecteru2/core/log"
	coreutils "github.com/projecteru2/core/utils"
)

// LabelMeta .
const LabelMeta = "ERU_META"

// HealthCheck .
type HealthCheck struct {
	TCPPorts []string
	HTTPPort string
	HTTPURL  string
	HTTPCode int
	Cmds     []string
}

type healthCheckMeta struct {
	Publish     []string
	HealthCheck *HealthCheck
}

// Guest yavirt virtual machine
type Guest struct {
	ID            string
	Status        string
	TransitStatus string
	CreateTime    int64
	TransitTime   int64
	UpdateTime    int64
	CPU           int
	Mem           int64
	Storage       int64
	ImageID       int64
	ImageName     string
	ImageUser     string
	Networks      map[string]string
	Labels        map[string]string
	IPs           []string
	Hostname      string
	Running       bool
	HealthCheck   *HealthCheck

	once sync.Once
}

// CheckHealth returns if the guest is healthy
func (g *Guest) CheckHealth(ctx context.Context, svc service.Service, timeout time.Duration, enableDefaultChecker bool) bool {
	logger := log.WithFunc("CheckHealth").WithField("ID", g.ID)
	// init health check bridge
	g.once.Do(func() {
		if meta, ok := g.Labels[LabelMeta]; ok {
			hcm := &healthCheckMeta{}
			err := json.Unmarshal([]byte(meta), hcm)
			if err != nil {
				logger.Error(ctx, err, "invalid json format, guest %v, meta %v", g.ID, meta)
				return
			}
			g.HealthCheck = hcm.HealthCheck
		}
		if enableDefaultChecker && g.HealthCheck == nil {
			// add a default checker if not exist
			g.HealthCheck = &HealthCheck{
				Cmds: []string{"whoami"},
			}
		}
	})

	logger.Debugf(ctx, "[eru agent] guest %v\n health check: %v", g, g.HealthCheck)
	if g.HealthCheck == nil {
		return true
	}

	var tcpChecker []string
	var httpChecker []string

	healthCheck := g.HealthCheck

	for _, port := range healthCheck.TCPPorts {
		for _, ip := range g.IPs {
			tcpChecker = append(tcpChecker, fmt.Sprintf("%s:%s", ip, port))
		}
	}
	if healthCheck.HTTPPort != "" {
		for _, ip := range g.IPs {
			httpChecker = append(httpChecker, fmt.Sprintf("http://%s:%s%s", ip, healthCheck.HTTPPort, healthCheck.HTTPURL)) //nolint
		}
	}

	f1 := utils.CheckHTTP(ctx, g.ID, httpChecker, healthCheck.HTTPCode, timeout)
	f2 := utils.CheckTCP(ctx, g.ID, tcpChecker, timeout)
	f3 := CheckCMD(ctx, svc, g.ID, healthCheck.Cmds, timeout)
	return f1 && f2 && f3
}

func CheckCMD(ctx context.Context, svc service.Service, ID string, cmdList []string, timeout time.Duration) bool {
	logger := log.WithFunc("CheckCMD").WithField("ID", ID)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for _, cmdStr := range cmdList {
		cmd := coreutils.MakeCommandLineArgs(cmdStr)
		ans := true
		utils.WithTimeout(ctx, timeout, func(ctx1 context.Context) {
			msg, err := svc.ExecuteGuest(ctx1, ID, cmd)
			if err != nil || msg.ExitCode != 0 {
				log.Warnf(ctx, "[checkHealth] guest %s execute cmd %s failed (err: %s, msg: %v)", ID, cmdStr, err, msg)
				ans = false
				return
			}
			logger.Debugf(ctx, "[checkHealth] guest %s execute cmd %s success, output: %v", ID, cmdStr, string(msg.Data))
		})
		if !ans {
			return ans
		}
	}
	return true
}
