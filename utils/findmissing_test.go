package utils

import (
	"slices"
	"testing"
)

func TestFindMissingItems(t *testing.T) {
	tests := []struct {
		name     string
		sliceOne []string
		sliceTwo []string
		want     []string
	}{
		{
			name:     "All present",
			sliceOne: []string{"a", "b", "c"},
			sliceTwo: []string{"a", "b"},
			want:     []string{},
		},
		{
			name:     "Some missing",
			sliceOne: []string{"a", "b", "c"},
			sliceTwo: []string{"a", "b", "d"},
			want:     []string{"d"},
		},
		{
			name:     "All missing",
			sliceOne: []string{"x", "y", "z"},
			sliceTwo: []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "Empty sliceTwo",
			sliceOne: []string{"a", "b", "c"},
			sliceTwo: []string{},
			want:     []string{},
		},
		{
			name:     "Empty sliceOne",
			sliceOne: []string{},
			sliceTwo: []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "Both empty",
			sliceOne: []string{},
			sliceTwo: []string{},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindMissingItems(tt.sliceOne, tt.sliceTwo)
			if !slices.Equal(got, tt.want) {
				t.Errorf("FindMissingItems() = %v, want %v", got, tt.want)
			}
		})
	}
}
