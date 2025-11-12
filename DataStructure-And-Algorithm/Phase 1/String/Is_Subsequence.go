package String

func isSubsequence(s string, t string) bool {
	i, j := 0, 0
	for i < len(s) && j < len(t) {
		if s[i] == t[j] {
			i++
		}
		j++
	}
	//如果 i == len(s)，说明 s 的每个字符都按顺序在 t 里找到了
	return i == len(s)
}
