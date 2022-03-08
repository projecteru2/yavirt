package device

import (
	"sync"

	"github.com/vishvananda/netlink"

	"github.com/projecteru2/yavirt/internal/errors"
)

// Driver .
type Driver struct {
	sync.Mutex

	*netlink.Handle
}

// New .
func New() (*Driver, error) {
	var handle, err = netlink.NewHandle()
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Driver{Handle: handle}, nil
}

func isFileExistsErr(err error) bool {
	return err.Error() == "file exists"
}
