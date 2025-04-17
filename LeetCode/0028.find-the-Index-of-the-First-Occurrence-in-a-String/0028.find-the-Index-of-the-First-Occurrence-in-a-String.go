package _028_find_the_Index_of_the_First_Occurrence_in_a_String

import "strings"

// 解法一：
func strStr(haystack string, needle string) int {
	return strings.Index(haystack, needle)
}

// 解法二：
func strStr1(haystack string, needle string) int {
	for i := 0; ; i++ {
		for j := 0; ; j++ {
			if j == len(needle) {
				return i
			}
			if i+j == len(haystack) {
				return -1
			}
			if needle[j] != haystack[j+i] {
				break
			}
		}
	}

}
