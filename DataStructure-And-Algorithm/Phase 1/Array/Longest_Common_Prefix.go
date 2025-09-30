package Array

import "sort"

func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	sort.Strings(strs) //按字典序升序排列
	start := strs[0]
	end := strs[len(strs)-1]
	res := ""

	for i := 0; i < len(start); i++ {
		if i < len(end) && start[i] == end[i] {
			res += string(start[i]) //start[i] 的类型是 byte
		} else {
			break
		}
	}
	return res
}
