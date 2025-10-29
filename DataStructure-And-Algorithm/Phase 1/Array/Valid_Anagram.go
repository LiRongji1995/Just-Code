package Array

func isAnagram(s string, t string) bool {
	if len(s) != len(t) {
		return false
	}

	count := make([]int, 26)

	for _, ch := range s {
		count[ch-'a']++
	}

	for _, ch := range t {
		count[ch-'a']--
	}

	for _, c := range count {
		if c != 0 {
			return false
		}
	}
	return true
}
