package hostdir

import (
	"fmt"
	"testing"

	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVolume(t *testing.T) {
	cases := []struct {
		in  string
		src string
		dst string
	}{
		{"/pool/image1:/data1", "/pool/image1", "/data1"},
		{"/pool/image1:/data1", "/pool/image1", "/data1"},
	}

	for _, c := range cases {
		vol, err := NewFromStr(c.in)
		assert.NilErr(t, err)

		assert.Equal(t, c.dst, vol.GetMountDir())
		assert.Equal(t, c.src, vol.Source)
	}
}

func TestGenerateXML(t *testing.T) {
	vol, err := NewFromStr("/pool/image1:/data1")
	vol.SetDevice("vda")
	require.Nil(t, err)
	bs, err := vol.GenerateXML()
	assert.NilErr(t, err)
	fmt.Printf("%s\n", string(bs))
}
