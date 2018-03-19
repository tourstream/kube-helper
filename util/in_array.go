package util

func Contains(haystack []string, needle string) bool {
	set := make(map[string]struct{}, len(haystack))
	for _, s := range haystack {
		set[s] = struct{}{}
	}

	_, ok := set[needle]
	return ok
}
