package utils

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	pkgutils "github.com/projecteru2/yavirt/pkg/utils"
)

func GenEndpointID() (string, error) {
	var uuid, err = pkgutils.UUIDStr()
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	return strings.ReplaceAll(uuid, "-", ""), nil
}

func GenDevName(prefix string) (string, error) {
	var endpID, err = GenEndpointID()
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	var name = fmt.Sprintf("%s%s", prefix, endpID[:pkgutils.Min(12, len(endpID))])
	return name, nil

}
