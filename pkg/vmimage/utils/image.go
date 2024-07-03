package utils

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func NormalizeImageName(fullname string) (user, name, tag string, err error) {
	var nameTag string
	switch parts := strings.Split(fullname, "/"); len(parts) {
	case 1:
		nameTag = parts[0]
	case 2:
		user, nameTag = parts[0], parts[1]
	default:
		err = fmt.Errorf("invalid image name: %s", fullname)
		return
	}

	switch parts := strings.Split(nameTag, ":"); len(parts) {
	case 1:
		name, tag = parts[0], "latest"
	case 2:
		name, tag = parts[0], parts[1]
	default:
		err = fmt.Errorf("invalid image name: %s", fullname)
		return
	}
	return
}

func CalcDigestOfFile(fname string) (string, error) {
	f, err := os.Open(fname)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()

	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// EnsureReaderClosed As the name says,
// blocks until the stream is empty, until we meet EOF
func EnsureReaderClosed(stream io.ReadCloser) error {
	if stream == nil {
		return nil
	}
	if _, err := io.Copy(io.Discard, stream); err != nil {
		return errors.Wrap(err, "Empty stream failed")
	}
	return stream.Close()
}

func ImageSize(_ context.Context, fname string) (int64, int64, error) {
	cmds := []string{"qemu-img", "info", "--output=json", fname}
	cmd := exec.Command(cmds[0], cmds[1:]...) //nolint
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to run qemu-img info")
	}
	res := map[string]any{}
	err = json.Unmarshal(output, &res)
	if err != nil {
		return 0, 0, errors.Wrap(err, "output is not json")
	}
	virtualSize := res["virtual-size"]
	actualSize := res["actual-size"]
	return int64(actualSize.(float64)), int64(virtualSize.(float64)), nil
}
