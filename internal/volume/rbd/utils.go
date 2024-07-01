package rbd

import (
	"fmt"
	"regexp"
)

func parseSnapName(input string) (string, string, string, error) {
	// Define the regular expression pattern for parsing the specified format
	regexPattern := `^([^/]+)/([^@]+)@(.+)$`

	// Compile the regular expression
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to compile regular expression: %v", err)
	}

	// Match the regular expression against the input
	matches := regex.FindStringSubmatch(input)
	if matches == nil || len(matches) != 4 {
		return "", "", "", fmt.Errorf("invalid format: %s", input)
	}

	// Extract matched components
	pool := matches[1]
	imgName := matches[2]
	snapName := matches[3]

	return pool, imgName, snapName, nil
}
