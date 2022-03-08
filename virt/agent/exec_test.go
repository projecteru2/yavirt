package agent

import (
	"testing"

	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/virt/agent/types"
)

func TestStdio(t *testing.T) {
	st := types.ExecStatus{Code: 0, Exited: true}
	_, _, err := st.Stdio()
	assert.NilErr(t, err)

	st = types.ExecStatus{Exited: false}
	_, _, err = st.Stdio()
	assert.Err(t, err)

	st = types.ExecStatus{Code: 0, Exited: true, Base64Out: "err"}
	_, _, err = st.Stdio()
	assert.Err(t, err)

	st = types.ExecStatus{Code: 1, Exited: true, Base64Err: "err"}
	_, _, err = st.Stdio()
	assert.Err(t, err)
}

func TestCheckStdio(t *testing.T) {
	st := types.ExecStatus{Code: 0, Exited: true}
	_, _, err := st.CheckStdio(func(_, _ []byte) bool { return false })
	assert.NilErr(t, err)

	st = types.ExecStatus{Code: 1, Exited: true}
	_, _, err = st.CheckStdio(func(_, _ []byte) bool { return true })
	assert.NilErr(t, err)

	st = types.ExecStatus{Code: 1, Exited: true}
	_, _, err = st.CheckStdio(func(_, _ []byte) bool { return false })
	assert.Err(t, err)
}
