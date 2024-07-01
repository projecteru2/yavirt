package device

import (
	"sync"

	"github.com/vishvananda/netlink"

	"github.com/cockroachdb/errors"
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
		return nil, errors.Wrap(err, "")
	}

	return &Driver{Handle: handle}, nil
}

func isFileExistsErr(err error) bool {
	return err.Error() == "file exists"
}
