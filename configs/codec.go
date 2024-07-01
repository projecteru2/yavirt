package configs

import (
	"bytes"

	"github.com/BurntSushi/toml"

	"github.com/cockroachdb/errors"
)

// Decode .
func Decode(raw string, conf *Config) error {
	if _, err := toml.Decode(raw, conf); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

// Encode .
func Encode(conf *Config, noIndents ...bool) (string, error) {
	var buf bytes.Buffer
	var enc = toml.NewEncoder(&buf)

	if len(noIndents) < 1 || !noIndents[0] {
		enc.Indent = "    "
	}

	if err := enc.Encode(conf); err != nil {
		return "", errors.Wrap(err, "")
	}

	return buf.String(), nil
}

// DecodeFile .
func DecodeFile(file string, conf *Config) (err error) {
	_, err = toml.DecodeFile(file, conf)
	return
}
