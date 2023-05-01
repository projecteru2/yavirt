package utils

import "encoding/json"

// JSONDecode .
func JSONDecode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// JSONEncode .
func JSONEncode(v any, indents ...string) ([]byte, error) {
	var indent string
	if len(indents) > 0 {
		indent = indents[0]
	}
	return json.MarshalIndent(v, "", indent)
}
