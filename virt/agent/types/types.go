package types

import (
	"encoding/base64"

	"github.com/projecteru2/yavirt/internal/errors"
)

// Diskfree .
type Diskfree struct {
	So              string
	Filesystem      string
	Blocks          int64
	UsedBlocks      int64
	AvailableBlocks int64
	UsedPercent     int
	Mount           string
}

// ExecStatus .
type ExecStatus struct {
	Exited       bool   `json:"exited"`
	Code         int    `json:"exitcode"`
	Base64Out    string `json:"out-data"`
	OutTruncated bool   `json:"out-truncated"`
	Base64Err    string `json:"err-data"`
	ErrTruncated bool   `json:"err-truncated"`

	Pid int
	Err error
}

// CheckStdio .
func (s ExecStatus) CheckStdio(check func([]byte, []byte) bool) (so, se []byte, err error) {
	if so, se, err = s.Stdio(); err == nil {
		// always ok while $? is 0
		return
	}

	if check(so, se) {
		err = nil
	}

	return
}

// Stdio .
func (s ExecStatus) Stdio() (so, se []byte, err error) {
	err = s.Error()

	var xe error
	if so, xe = s.stdout(); xe != nil {
		return nil, nil, errors.Wrap(err, xe)
	}

	if se, xe = s.stderr(); xe != nil {
		return nil, nil, errors.Wrap(err, xe)
	}

	return
}

func (s ExecStatus) stdout() ([]byte, error) {
	return base64.StdEncoding.DecodeString(s.Base64Out)
}

func (s ExecStatus) stderr() ([]byte, error) {
	return base64.StdEncoding.DecodeString(s.Base64Err)
}

// CheckReturnCode .
func (s ExecStatus) CheckReturnCode() (bool, error) {
	if err := s.Error(); err != nil && !errors.Contain(err, errors.ErrExecNonZeroReturn) {
		return false, errors.Trace(err)
	}
	return s.Code == 0, nil
}

func (s ExecStatus) Error() error {
	switch {
	case s.Err != nil:
		return errors.Trace(s.Err)

	case !s.Exited:
		return errors.ErrExecIsRunning

	case s.Code != 0:
		return errors.Annotatef(errors.ErrExecNonZeroReturn,
			"return %d; stdout: %s; stderr: %s",
			s.Code, decodeToString(s.Base64Out), decodeToString(s.Base64Err))

	default:
		return nil
	}
}

func decodeToString(src string) string {
	decodeString, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return src
	}
	return string(decodeString)
}
