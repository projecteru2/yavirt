package resources

import (
	"os/exec"
	"path"
	"strings"

	"github.com/dustin/go-humanize"
	stotypes "github.com/projecteru2/resource-storage/storage/types"
)

type StorageManager struct {
	sto *stotypes.NodeResource
}

func newStorageManager() (*StorageManager, error) {
	sto, err := FetchStorage()
	if err != nil {
		return nil, err
	}
	return &StorageManager{
		sto: sto,
	}, nil
}

func FetchStorage() (*stotypes.NodeResource, error) {
	ans := &stotypes.NodeResource{}
	total := int64(0)
	vols := stotypes.Volumes{}
	// use df to fetch volume information

	cmdOut, err := exec.Command("df", "-h").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(cmdOut), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) != 6 {
			continue
		}
		var size uint64
		size, err = humanize.ParseBytes(parts[1])
		if err != nil {
			return nil, err
		}
		mountPoint := parts[len(parts)-1]
		if path.Base(mountPoint) == "eru" {
			vols[mountPoint] = int64(size)
			total += int64(size)
		}
	}
	ans.Volumes = vols
	ans.Storage = total
	return ans, nil
}
