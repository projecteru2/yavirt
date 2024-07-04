package resources

import (
	"context"
	"path"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/dustin/go-humanize"
	stotypes "github.com/projecteru2/resource-storage/storage/types"
	"github.com/projecteru2/yavirt/pkg/sh"
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

	so, se, err := sh.ExecInOut(context.TODO(), nil, nil, "df", "-h")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run df: %s", string(se))
	}
	lines := strings.Split(string(so), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) != 6 {
			continue
		}
		var size uint64
		size, err = humanize.ParseBytes(parts[1])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse size %s", parts[1])
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
