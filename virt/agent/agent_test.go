package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/projecteru2/yavirt/test/assert"
	"github.com/projecteru2/yavirt/test/mock"
	"github.com/projecteru2/yavirt/virt/agent/mocks"
	"github.com/projecteru2/yavirt/virt/agent/types"
)

func TestAgent(t *testing.T) {
	var agent = New("/tmp/virt/sock/guest-000001.sock")
	var in = "ping"
	var out = []byte("pong")

	var ret = types.ExecStatus{
		Exited:    true,
		Base64Out: base64.StdEncoding.EncodeToString(out),
	}

	enc, err := json.Marshal(ret)
	assert.NilErr(t, err)

	var qmp = &mocks.Qmp{}
	defer qmp.AssertExpectations(t)
	qmp.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return([]byte(`{"pid":6735}`), nil).Once()
	qmp.On("ExecStatus", 6735).Return(enc, nil).Once()

	agent.qmp = qmp
	var st = <-agent.ExecOutput(context.Background(), in)
	assert.NotNil(t, st)
	assert.NilErr(t, st.Error())
	assert.Equal(t, 0, st.Code)

	so, se, err := st.Stdio()
	assert.NilErr(t, err)
	assert.Equal(t, out, so)
	assert.Equal(t, 0, len(se))
}

func TestFileReader(t *testing.T) {
	if os.Getenv("REAL") != "1" {
		return
	}

	var agent = New("/opt/yavirtd/sock/00000000010160627254733565100003.sock")
	rd, err := OpenFile(agent, "/tmp/snmpss.cache", "r")
	assert.NilErr(t, err)

	defer rd.Close()

	p := make([]byte, 10)
	n, err := rd.Read(p)
	assert.NilErr(t, err)
	assert.Equal(t, 10, n)
	t.Logf(" read /tmp/snmpss.cache: %s ", string(p))

	n, err = rd.Read(p)
	assert.NilErr(t, err)
	assert.Equal(t, 10, n)
	t.Logf(" read /tmp/snmpss.cache: %s ", string(p))

	n, err = rd.Read(p)
	assert.NilErr(t, err)
	assert.Equal(t, 9, n)
	t.Logf(" read /tmp/snmpss.cache: %s ", string(p))
}
