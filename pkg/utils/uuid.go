package utils

import (
	"github.com/google/uuid"

	"github.com/cockroachdb/errors"
)

// UUIDStr .
func UUIDStr() (string, error) {
	var u, err = uuid.NewUUID()
	if err != nil {
		return "", errors.Wrap(err, "NewUUID error")
	}
	return u.String(), nil
}

// CheckUUID .
func CheckUUID(raw string) (err error) {
	_, err = uuid.Parse(raw)
	return
}
