package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeImageName(t *testing.T) {
	tests := []struct {
		fullname      string
		expectedUser  string
		expectedName  string
		expectedTag   string
		expectedError string
	}{
		{"ubuntu", "", "ubuntu", "latest", ""},
		{"myuser/myimage:1.0", "myuser", "myimage", "1.0", ""},
		{"myimage:2.0", "", "myimage", "2.0", ""},
		{"invalid/image/name:tag:extra", "", "", "", "invalid image name: invalid/image/name:tag:extra"},
		{"invalid/image/name:tag", "", "", "", "invalid image name: invalid/image/name:tag"},
		{"image/name:tag:extra", "", "", "", "invalid image name: image/name:tag:extra"},
	}

	for _, test := range tests {
		user, name, tag, err := NormalizeImageName(test.fullname)

		if test.expectedError == "" {
			assert.Nil(t, err, "Error should be nil")
		} else {
			assert.EqualError(t, err, test.expectedError, "Error message should match")
			continue
		}
		assert.Equal(t, test.expectedUser, user, "User should match")
		assert.Equal(t, test.expectedName, name, "Name should match")
		assert.Equal(t, test.expectedTag, tag, "Tag should match")

	}
}
