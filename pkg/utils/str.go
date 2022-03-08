package utils

import "strings"

// LowerLetters .
var LowerLetters = lowerLetters()

// MergeStrings .
func MergeStrings(a, b []string) []string {
	var dic = make(map[string]struct{})
	var strs = make([]string, 0, len(a)+len(b))

	var appendx = func(ar []string) {
		for _, k := range ar {
			if _, exists := dic[k]; exists {
				continue
			}
			dic[k] = struct{}{}
			strs = append(strs, k)
		}
	}

	appendx(a)
	appendx(b)

	return strs
}

// PartLeft partitions the str by the sep.
func PartLeft(str, sep string) (string, string) {
	switch i := strings.Index(str, sep); {
	case i < 0:
		return str, ""
	case i == 0:
		return "", str[i+1:]
	default:
		return str[:i], str[i+1:]
	}
}

// PartRight partitions the str by the sep.
func PartRight(str, sep string) (string, string) {
	switch i := strings.LastIndex(str, sep); {
	case i < 0:
		return "", str
	case i >= len(str)-1:
		return str[:i], ""
	default:
		return str[:i], str[i+1:]
	}
}

func lowerLetters() string {
	var buf = make([]byte, 26) //nolint:gomnd // 26 alphabets
	for i := range buf {
		buf[i] = 'a' + byte(i)
	}
	return string(buf)
}
