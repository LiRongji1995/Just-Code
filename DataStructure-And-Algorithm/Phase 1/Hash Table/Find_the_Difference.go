package Hash_Table

func findTheDifference(s string, t string) byte {
	freq := make(map[byte]int)

	for i := range len(t) {
		freq[t[i]]++
	}

	for i := range len(s) {
		freq[s[i]]--
	}

	for ch, cnt := range freq {
		if cnt > 0 {
			return ch
		}
	}
	return 0

}
