package etcd

import (
	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/util"
)

func encode(v interface{}) (string, error) { //nolint
	var buf, err = util.JSONEncode(v, "\t")
	if err != nil {
		return "", errors.Trace(err)
	}
	return string(buf), nil
}

func decode(data []byte, v interface{}) error {
	return util.JSONDecode(data, v)
}
