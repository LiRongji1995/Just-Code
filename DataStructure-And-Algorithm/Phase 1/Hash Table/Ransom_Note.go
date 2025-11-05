package Hash_Table

func canConstruct(ransomNote string, magazine string) bool {
	/*
		统计 magazine 里每个字符的库存，再去扣 ransomNote 里的需求，看会不会不够

		1.准备一个 map[rune]int，key 是字符，value 是这个字符的数量。

		2.先遍历 magazine，把每个字符的数量记进去（进货）。

		3.再遍历 ransomNote，每来一个字符，就把这个字符的数量减 1（出货）。

		4.如果减完发现某个字符数量 < 0，说明 magazine 不够用，直接返回 false。

		5.遍历完都没问题，就返回 true。
	*/
	freq := make(map[rune]int)

	for _, ch := range magazine {
		freq[ch]++
	}

	for _, ch := range ransomNote {
		freq[ch]--
		if freq[ch] < 0 {
			return false
		}
	}
	return true
}
