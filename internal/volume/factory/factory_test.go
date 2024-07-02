package factory

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/volume/local"
	"github.com/projecteru2/yavirt/internal/volume/rbd"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/store"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func TestLoadVolumes(t *testing.T) {
	idgen.Setup(111)
	err := store.Setup(configs.Conf, t)
	assert.NilErr(t, err)
	v1Str := "/src:/dst:rw:1G:1:2:3:4"
	v2Str := "pool/image:/dst:rw:1G:1:2:3:4"
	v1, err := local.NewVolumeFromStr(v1Str)
	assert.NilErr(t, err)
	v2, err := rbd.NewFromStr(v2Str)
	assert.NilErr(t, err)
	v1.GenerateID()
	v2.GenerateID()
	err = v1.Save()
	assert.NilErr(t, err)
	err = v2.Save()
	assert.NilErr(t, err)

	vols, err := LoadVolumes([]string{v1.ID, v2.ID})
	assert.NilErr(t, err)
	assert.Equal(t, 2, len(vols))

	vv1, ok := vols[0].(*local.Volume)
	assert.True(t, ok)
	assert.Equal(t, v1.ID, vv1.ID)
	assert.Equal(t, "/src", vv1.Source)
	assert.Equal(t, "/dst", vv1.Destination)
	assert.Equal(t, utils.GB, vv1.SizeInBytes)
	assert.Equal(t, int64(1), vv1.ReadIOPS)
	assert.Equal(t, int64(2), vv1.WriteIOPS)
	assert.Equal(t, int64(3), vv1.ReadBPS)
	assert.Equal(t, int64(4), vv1.WriteBPS)

	vv2, ok := vols[1].(*rbd.Volume)
	assert.True(t, ok)
	assert.Equal(t, v2.ID, vv2.ID)
	assert.Equal(t, "pool", vv2.Pool)
	assert.Equal(t, "image", vv2.Image)
	assert.Equal(t, "/dst", vv2.Destination)
	assert.Equal(t, utils.GB, vv1.SizeInBytes)
	assert.Equal(t, int64(1), vv2.ReadIOPS)
	assert.Equal(t, int64(2), vv2.WriteIOPS)
	assert.Equal(t, int64(3), vv2.ReadBPS)
	assert.Equal(t, int64(4), vv2.WriteBPS)
}

func TestVolumes(t *testing.T) {
	idgen.Setup(111)
	err := store.Setup(configs.Conf, t)
	assert.NilErr(t, err)
	var ids []string
	for i := 0; i < 5; i++ {
		v1Str := fmt.Sprintf("/src%d:/dst%d:rw:1G", i, 2*i)
		v2Str := fmt.Sprintf("pool/image%d:/dst%d:rw:1G", i, 2*i+1)
		v1, err := local.NewVolumeFromStr(v1Str)
		assert.NilErr(t, err)
		v2, err := rbd.NewFromStr(v2Str)
		assert.NilErr(t, err)
		v1.GenerateID()
		v2.GenerateID()
		err = v1.Save()
		assert.NilErr(t, err)
		err = v2.Save()
		assert.NilErr(t, err)
		ids = append(ids, v1.ID)
		ids = append(ids, v2.ID)
	}
	vols, err := LoadVolumes(ids)
	assert.Equal(t, 10, vols.Len())
	assert.Equal(t, ids, vols.IDs())
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	randID := ids[rand.Intn(len(ids))]
	vol, err := vols.Find(randID)
	assert.NilErr(t, err)
	assert.True(t, vols.Exists(vol.GetMountDir()))

	vol, err = vols.GetMntVol("/dst9/haha")
	assert.NilErr(t, err)
	assert.Equal(t, "/dst9", vol.GetMountDir())

	vol, err = vols.GetMntVol("/dst10/haha")
	assert.NilErr(t, err)
	assert.Nil(t, vol)
}
