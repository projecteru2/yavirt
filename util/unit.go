package util

const (
	// Byte .
	Byte int64 = 1 << (10 * iota) //nolint:gomnd // 10 * iota plus shift left simple math
	// KB .
	KB
	// MB .
	MB
	// GB .
	GB
	// TB .
	TB
)

// ConvToMB .
func ConvToMB(b int64) int64 {
	return b / MB
}
