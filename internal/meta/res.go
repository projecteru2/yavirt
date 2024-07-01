package meta

import (
	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Resource .
type Resource interface {
	MetaKey() string
	GetVer() int64
	SetVer(int64)
	IncrVer()
}

// Resources .
type Resources []Resource

// Concate .
func (res *Resources) Concate(b Resources) {
	*res = append(*res, b...)
}

// Encode .
func (res Resources) Encode() (map[string]string, error) {
	var data = map[string]string{}

	for _, r := range res {
		var enc, err = utils.JSONEncode(r, "\t")
		if err != nil {
			return nil, errors.Wrapf(err, "encode resource %v failed", r)
		}

		data[r.MetaKey()] = string(enc)
	}

	return data, nil
}

// IncrVer .
func (res Resources) IncrVer() {
	for _, r := range res {
		r.IncrVer()
	}
}

// Vers .
func (res Resources) Vers() map[string]int64 {
	var vers = map[string]int64{}
	for _, r := range res {
		vers[r.MetaKey()] = r.GetVer()
	}
	return vers
}
