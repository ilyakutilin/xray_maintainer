package utils

// FindMissingItems returns items in sliceTwo that are not present in sliceOne.
func FindMissingItems(sliceOne, sliceTwo []string) []string {
	// Simulate a set using a map. We use struct and not bool for memory efficiency.
	set := make(map[string]struct{})
	for _, val := range sliceOne {
		set[val] = struct{}{}
	}

	var missing []string
	for _, val := range sliceTwo {
		if _, found := set[val]; !found {
			missing = append(missing, val)
		}
	}

	return missing
}

// HasDuplicates checks if a slice contains any duplicate strings.
func FindDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	duplicates := make(map[string]bool)
	for _, str := range slice {
		if seen[str] {
			duplicates[str] = true
		} else {
			seen[str] = true
		}
	}

	// Convert map keys to slice
	result := make([]string, 0, len(duplicates))
	for dup := range duplicates {
		result = append(result, dup)
	}
	return result
}
