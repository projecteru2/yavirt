package config

import (
	"fmt"
	"time"
)

// Duration .
type Duration time.Duration

// Duration .
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// UnmarshalText .
func (d *Duration) UnmarshalText(text []byte) error {
	var dur, err = time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// MarshalText .
func (d Duration) MarshalText() ([]byte, error) {
	if d == 0 {
		return []byte("0"), nil
	}

	var dur = time.Duration(d)
	if dur < 0 {
		dur = -dur
	}

	switch {
	case dur%time.Hour == 0:
		return []byte(fmt.Sprintf("%dh", dur/time.Hour)), nil
	case dur%time.Minute == 0:
		return []byte(fmt.Sprintf("%dm", dur/time.Minute)), nil
	case dur%time.Second == 0:
		return []byte(fmt.Sprintf("%ds", dur/time.Second)), nil
	case dur%time.Millisecond == 0:
		return []byte(fmt.Sprintf("%dms", dur/time.Millisecond)), nil
	default:
		return []byte(dur.String()), nil
	}
}
