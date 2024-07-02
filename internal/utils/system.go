package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/projecteru2/core/log"
	"github.com/shirou/gopsutil/process"
)

func EnforceRoot() {
	// Make sure the command is run with super user priviladges
	if os.Getuid() != 0 {
		fmt.Println("Need super user privileges: Operation not permitted")
		os.Exit(1)
	}
}
func PSContains(proc []string, procList []*process.Process) bool {
	logger := log.WithFunc("psContains")
	for _, p := range procList {
		cmds, err := p.CmdlineSlice()
		if err != nil {
			// Failed to get CLI arguments for this process.
			// Maybe it doesn't exist any more - move on to the next one.
			logger.Debugf(context.TODO(), "Error getting CLI arguments: %s", err)
			continue
		}
		var match bool
		for i, p := range proc {
			if i >= len(cmds) {
				break
			} else if cmds[i] == p {
				match = true
			}
		}

		// If we got a match, return true. Otherwise, try the next
		// process in the list.
		if match {
			return true
		}
	}
	return false
}
