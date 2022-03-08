package util

import (
	"github.com/google/uuid"

	"github.com/projecteru2/yavirt/internal/errors"
)

// UUIDStr .
func UUIDStr() (string, error) {
	var u, err = uuid.NewUUID()
	if err != nil {
		return "", errors.Trace(err)
	}
	return u.String(), nil
}

// CheckUUID .
func CheckUUID(raw string) (err error) {
	_, err = uuid.Parse(raw)
	return
}
