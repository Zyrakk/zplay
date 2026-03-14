package util

import "testing"

func TestMemoryToMi(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"4Gi", 4096, false},
		{"1Gi", 1024, false},
		{"512Mi", 512, false},
		{"8Gi", 8192, false},
		{"256Mi", 256, false},
		{"", 0, true},
		{"4", 0, true},
		{"4Xi", 0, true},
		{"abcGi", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := MemoryToMi(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("MemoryToMi(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("MemoryToMi(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestInferMemoryLimit(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"4Gi", "8Gi", false},
		{"1Gi", "2Gi", false},
		{"512Mi", "1Gi", false},
		{"256Mi", "512Mi", false},
		{"3Gi", "6Gi", false},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := InferMemoryLimit(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferMemoryLimit(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("InferMemoryLimit(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
