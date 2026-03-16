package main

import (
	"testing"
	"time"
)

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		contains string
	}{
		{"minutes", time.Now().Add(-30 * time.Minute), "m"},
		{"hours", time.Now().Add(-3 * time.Hour), "h"},
		{"days", time.Now().Add(-48 * time.Hour), "d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUptime(tt.start)
			if result == "" || result == "N/A" {
				t.Errorf("formatUptime returned empty/NA for %s", tt.name)
			}
		})
	}
}

func TestFormatMemoryGi(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{1073741824, "1.0Gi"},
		{4294967296, "4.0Gi"},
		{536870912, "512Mi"},
		{268435456, "256Mi"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatMemoryGi(tt.bytes)
			if got != tt.expected {
				t.Errorf("formatMemoryGi(%d) = %q, want %q", tt.bytes, got, tt.expected)
			}
		})
	}
}

func TestFormatCPUCores(t *testing.T) {
	tests := []struct {
		milliCores int64
		expected   string
	}{
		{500, "0.5"},
		{1000, "1.0"},
		{2000, "2.0"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatCPUCores(tt.milliCores)
			if got != tt.expected {
				t.Errorf("formatCPUCores(%d) = %q, want %q", tt.milliCores, got, tt.expected)
			}
		})
	}
}
