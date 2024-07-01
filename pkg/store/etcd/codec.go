package etcd

import (
	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func encode(v any) (string, error) { //nolint
	var buf, err = utils.JSONEncode(v, "\t")
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	return string(buf), nil
}

func decode(data []byte, v any) error {
	return utils.JSONDecode(data, v)
}
