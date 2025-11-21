package langlang

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRange_Contains(t *testing.T) {
	tests := []struct {
		name     string
		parent   Range
		other    Range
		expected bool
	}{
		{
			name:     "fully contained range",
			parent:   NewRange(0, 10),
			other:    NewRange(2, 8),
			expected: true,
		},
		{
			name:     "identical ranges",
			parent:   NewRange(5, 15),
			other:    NewRange(5, 15),
			expected: true,
		},
		{
			name:     "other starts at same position",
			parent:   NewRange(0, 10),
			other:    NewRange(0, 5),
			expected: true,
		},
		{
			name:     "other ends at same position",
			parent:   NewRange(0, 10),
			other:    NewRange(5, 10),
			expected: true,
		},
		{
			name:     "other starts before parent",
			parent:   NewRange(5, 15),
			other:    NewRange(3, 10),
			expected: false,
		},
		{
			name:     "other ends after parent",
			parent:   NewRange(5, 15),
			other:    NewRange(10, 20),
			expected: false,
		},
		{
			name:     "other completely before parent",
			parent:   NewRange(10, 20),
			other:    NewRange(0, 5),
			expected: false,
		},
		{
			name:     "other completely after parent",
			parent:   NewRange(0, 10),
			other:    NewRange(15, 25),
			expected: false,
		},
		{
			name:     "other overlaps start boundary",
			parent:   NewRange(5, 15),
			other:    NewRange(3, 8),
			expected: false,
		},
		{
			name:     "other overlaps end boundary",
			parent:   NewRange(5, 15),
			other:    NewRange(12, 18),
			expected: false,
		},
		{
			name:     "other completely encompasses parent",
			parent:   NewRange(5, 15),
			other:    NewRange(0, 20),
			expected: false,
		},
		{
			name:     "zero-length range at start",
			parent:   NewRange(0, 10),
			other:    NewRange(0, 0),
			expected: true,
		},
		{
			name:     "zero-length range at end",
			parent:   NewRange(0, 10),
			other:    NewRange(10, 10),
			expected: true,
		},
		{
			name:     "zero-length range in middle",
			parent:   NewRange(0, 10),
			other:    NewRange(5, 5),
			expected: true,
		},
		{
			name:     "zero-length range before parent",
			parent:   NewRange(5, 10),
			other:    NewRange(3, 3),
			expected: false,
		},
		{
			name:     "zero-length range after parent",
			parent:   NewRange(5, 10),
			other:    NewRange(12, 12),
			expected: false,
		},
		{
			name:     "both zero-length at same position",
			parent:   NewRange(5, 5),
			other:    NewRange(5, 5),
			expected: true,
		},
		{
			name:     "parent zero-length, other has length",
			parent:   NewRange(5, 5),
			other:    NewRange(5, 10),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.parent.Contains(tt.other)
			assert.Equal(t, tt.expected, result,
				"Range(%d..%d).Contains(%d..%d) should be %v",
				tt.parent.Start, tt.parent.End,
				tt.other.Start, tt.other.End,
				tt.expected)
		})
	}
}
