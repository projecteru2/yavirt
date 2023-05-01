package utils

import (
	"crypto/rand"
	"fmt"

	"github.com/projecteru2/yavirt/pkg/errors"
)

// QemuMAC .
func QemuMAC() (string, error) {
	var buf, err = RandBuf(3)
	if err != nil {
		return "", errors.Trace(err)
	}
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", buf[0], buf[1], buf[2]), nil
}

// RandBuf .
func RandBuf(n int) ([]byte, error) {
	var buf = make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return nil, errors.Trace(err)
	}
	return buf, nil
}
