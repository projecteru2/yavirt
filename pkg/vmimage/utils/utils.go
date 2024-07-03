package utils

import (
	"errors"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func IPReachable(ip string, timeout time.Duration) error {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return err
	}
	pinger.Timeout = timeout
	pinger.Count = 2
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		return err
	}
	stats := pinger.Statistics()
	if stats.PacketsRecv <= 0 {
		return errors.New("unreachable")
	}
	return nil
}
