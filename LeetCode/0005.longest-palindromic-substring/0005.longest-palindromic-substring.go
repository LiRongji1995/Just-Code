package _0005_longest_palindromic_substring

// 解法⼀ 动态规划;状态转移方程 dp[i][j] = (s[i] == s[j]) && ((j - i < 3) || dp[i + 1][j - 1])
//dp[i][j] 表示字符串 s 从索引 i 到索引 j 这一段是否是回文子串
//s[i] == s[j]这个条件检查 s[i] 和 s[j] 这两个字符是否相等
//j - i < 3这个条件是用来处理边界情况的
//如果 j - i < 3（比如 "aa" 或 "aba"），只要 s[i] == s[j]，那么 s[i...j] 一定是回文串
//dp[i + 1][j - 1]这是用于检查更短的子串 s[i+1...j-1] 是否是回文
func longestPalindrome1(s string) string {
	res, dp := "", make([][]bool, len(s))
	for i := 0; i < len(s); i++ {
		dp[i] = make([]bool, len(s))
	}
	for i := len(s) - 1; i >= 0; i-- { // // i 从右向左遍历
		for j := i; j < len(s); j++ { // j 从 i 开始向右遍历
			dp[i][j] = (s[i] == s[j]) && ((j-i < 3) || dp[i+1][j-1])
			if dp[i][j] && (res == "" || j-i+1 > len(res)) {
				res = s[i : j+1]
			}
		}
	}
	return res
}
