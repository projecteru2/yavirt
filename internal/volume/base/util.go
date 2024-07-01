package base

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/virt/nic"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func ResetFstab(gfs guestfs.Guestfs) error {
	origFstabEntries, err := gfs.GetFstabEntries()
	if err != nil {
		return errors.Wrap(err, "")
	}

	blkids, err := gfs.GetBlkids()
	if err != nil {
		return errors.Wrap(err, "")
	}

	var cont string
	for dev, entry := range origFstabEntries {
		if blkids.Exists(dev) {
			cont += fmt.Sprintf("%s\n", strings.TrimSpace(entry))
		}
	}

	return gfs.Write(types.FstabFile, cont)
}

func ResetUserImage(gfs guestfs.Guestfs) error {
	if err := ResetFstab(gfs); err != nil {
		return errors.Wrap(err, "")
	}

	return resetEth0(gfs)
}

func resetEth0(gfs guestfs.Guestfs) error {
	distro, err := gfs.Distro()
	if err != nil {
		return errors.Wrap(err, "")
	}

	path, err := nic.GetEthFile(distro, "eth0")
	if err != nil {
		return errors.Wrap(err, "")
	}
	return gfs.Remove(path)
}

// GetDevicePathByName .
func GetDevicePathByName(name string) string {
	return filepath.Join("/dev", name)
}

// GetDeviceName .
func GetDeviceName(sn int) string {
	return fmt.Sprintf("vd%s", string(utils.LowerLetters[sn]))
}

func GetDevicePathBySerialNumber(sn int) string {
	return GetDevicePathByName(GetDeviceName(sn))
}

// Resize root partition
// mainly used to expand system volumn
func ResizeRootPartition(ctx context.Context, ga agent.Interface, devPath string) error {
	var st = <-ga.ExecOutput(ctx, "df")
	so, se, err := st.Stdio()
	if err != nil {
		return errors.Wrapf(err, "run `df` failed: %s", string(se))
	}
	lines := strings.Split(string(so), "\n")
	rootDev := ""
	pidx := "" // partiton index
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) != 6 {
			continue
		}
		if parts[5] == "/" {
			rootDev = parts[0]
		}
	}
	if len(rootDev) > 0 {
		idx := len(rootDev) - 1
		for idx >= 0 {
			if rootDev[idx] >= '0' && rootDev[idx] <= '9' {
				idx--
			}
		}
		devPath = rootDev[:idx+1]
		pidx = rootDev[idx+1:]
	}
	if len(rootDev) == 0 || len(pidx) == 0 {
		return errors.Errorf("Can't find root dev or sn: %s", rootDev)
	}
	var cmds = [][]string{
		// {"parted", "-s", devPath, "resizepart", pidx, "100%"},
		// Just need to run `parted devPath resizepart pidx 100%`, but parted prompt,so use pipeline here
		{"bash", "-c", fmt.Sprintf("echo 'Y' | sudo parted ---pretend-input-tty %s resizepart %s %s", devPath, pidx, "100%")},
		{"resize2fs", rootDev},
	}
	return ExecCommands(ctx, ga, cmds)
}

func ExecCommands(ctx context.Context, ga agent.Interface, cmds [][]string) error {
	for _, args := range cmds {
		var st = <-ga.ExecOutput(ctx, args[0], args[1:]...)
		if err := st.Error(); err != nil {
			return errors.Wrapf(err, "%v", args)
		}
	}
	return nil
}

func StopSystemdService(ctx context.Context, ga agent.Interface, serviceName string) error {
	var st = <-ga.Exec(ctx, "systemctl", "stop", serviceName)
	if err := st.Error(); err != nil {
		return errors.Wrapf(err, "systemctl stop %s failed", serviceName)
	}

	return nil
}

func RestartSystemdServices(ctx context.Context, ga agent.Interface, stoppedServices []string) error {
	for _, serviceName := range stoppedServices {
		var st = <-ga.Exec(ctx, "systemctl", "start", serviceName)
		if err := st.Error(); err != nil {
			return errors.Wrapf(err, "systemctl start %s failed", serviceName)
		}
	}
	return nil
}

func StopSystemdServices(ctx context.Context, ga agent.Interface, devPath string) ([]string, error) {
	var st = <-ga.ExecOutput(ctx, "fuser", "-m", devPath)
	so, se, err := st.Stdio()
	if err != nil && (len(so) > 0 || len(se) > 0) { // Fuser return status code 1 if no process running
		return nil, errors.Wrapf(err, "fuser on %s failed", devPath)
	}

	re := regexp.MustCompile(`[0-9]+`)
	pids := re.FindAllString(string(so), -1)

	var stoppedServices []string
	for _, pid := range pids {
		switch serviceName, err := findService(ctx, ga, pid); {
		case err != nil:
			return nil, errors.Wrap(err, "")

		case len(serviceName) > 0:
			if err := StopSystemdService(ctx, ga, serviceName); err != nil {
				return nil, errors.Wrap(err, "")
			}
			stoppedServices = append(stoppedServices, serviceName)

		default:
			continue
		}
	}

	return stoppedServices, nil
}

func findService(ctx context.Context, ga agent.Interface, pid string) (string, error) {
	for {
		switch name, se := getServiceNameByPid(ctx, ga, pid); {
		case strings.HasPrefix(se, "Failed "): // Doesn't exist systemd unit with this pid
			ppid, err := getPpid(ctx, ga, pid)
			if err != nil {
				return "", errors.Wrap(err, "")
			}
			pid = ppid

		case len(name) > 0:
			switch valid, err := isService(ctx, ga, name); {
			case err != nil:
				return "", errors.Wrap(err, "")

			case valid:
				return name, nil

			default: // unit with this pid exist but not service type
				return "", nil
			}

		default:
			return "", nil
		}
	}
}

func getPpid(ctx context.Context, ga agent.Interface, pid string) (string, error) {
	var st = <-ga.ExecOutput(ctx, "ps", "--ppid", pid)
	so, _, err := st.Stdio()
	if err != nil {
		return "", errors.Wrapf(err, "find ppid for %s failed", pid)
	}
	if len(so) == 0 {
		return "", errors.Newf("ppid for %s is empty", pid)
	}
	return string(so), nil
}

func getServiceNameByPid(ctx context.Context, ga agent.Interface, pid string) (string, string) {
	var st = <-ga.ExecOutput(ctx, "systemctl", "status", pid)
	so, se, err := st.Stdio()
	if err != nil {
		return "", string(se) // No service with this pid or service is stopped/failed
	}
	soSplit := strings.Fields(string(so))
	if len(soSplit) < 2 || len(soSplit[1]) == 0 {
		return "", ""
	}
	return soSplit[1], string(se)
}

func isService(ctx context.Context, ga agent.Interface, unitName string) (bool, error) {
	var st = <-ga.ExecOutput(ctx, "systemctl", "list-units", "--all", "-t", "service",
		"--full", "--no-legend", unitName)

	so, se, err := st.Stdio()
	if err != nil {
		if len(so) > 0 || len(se) > 0 {
			return false, errors.Wrapf(err, "systemctl check service %s failed", unitName)
		}
		return false, nil // Not found service with name unitName but not considered as error
	}
	soSplit := strings.Fields(string(so))
	if len(soSplit) < 1 {
		return false, errors.Newf("systemctl check service %s wrong output", unitName)
	}

	return true, nil
}
