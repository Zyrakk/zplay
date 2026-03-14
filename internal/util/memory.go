package util

import (
	"fmt"
	"strconv"
)

// MemoryToMi converts a Kubernetes memory string (e.g. "4Gi", "512Mi") to mebibytes.
func MemoryToMi(memory string) (int, error) {
	if len(memory) < 3 {
		return 0, fmt.Errorf("invalid memory format: %s", memory)
	}

	value, err := strconv.Atoi(memory[:len(memory)-2])
	if err != nil {
		return 0, fmt.Errorf("invalid memory format: %s", memory)
	}

	switch memory[len(memory)-2] {
	case 'G':
		return value * 1024, nil
	case 'M':
		return value, nil
	default:
		return 0, fmt.Errorf("invalid memory format: %s", memory)
	}
}

// InferMemoryLimit returns a memory limit that is double the given request.
func InferMemoryLimit(memory string) (string, error) {
	memoryMi, err := MemoryToMi(memory)
	if err != nil {
		return "", err
	}
	limitMi := memoryMi * 2
	if limitMi%1024 == 0 {
		return fmt.Sprintf("%dGi", limitMi/1024), nil
	}
	return fmt.Sprintf("%dMi", limitMi), nil
}
