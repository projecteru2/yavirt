package rbd

import (
	"fmt"
	"testing"

	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func TestNewVolume(t *testing.T) {
	cases := []struct {
		in   string
		src  string
		dst  string
		size int64
	}{
		{"pool/image1:/data1", "pool/image1", "/data1", int64(0)},
		{"pool/image1:/data1:rw:2G", "pool/image1", "/data1", int64(2 * utils.GB)},
	}

	for _, c := range cases {
		vol, err := NewFromStr(c.in)
		assert.NilErr(t, err)

		assert.Equal(t, c.dst, vol.GetMountDir())
		assert.Equal(t, c.src, vol.GetSource())
		assert.Equal(t, c.size, vol.SizeInBytes)
	}
}

func TestGenerateXML(t *testing.T) {
	vol, err := NewFromStr("pool/image1:/data1:rw:2G")
	vol.SetDevice("vda")
	assert.NilErr(t, err)
	bs, err := vol.GenerateXML()
	assert.NilErr(t, err)
	fmt.Printf("%s\n", string(bs))
}
