package String

func strStr(haystack string, needle string) int {
	if len(needle) == 0 {
		return 0
	}

	m, n := len(haystack), len(needle)
	if m < n {
		return -1
	}

	for i := 0; i <= m-n; i++ {
		j := 0
		for j < n && haystack[i+j] == needle[j] {
			j++
		}
		if j == n {
			return i
		}
	}
	return -1
}
