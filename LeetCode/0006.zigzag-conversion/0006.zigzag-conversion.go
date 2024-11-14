package _006_zigzag_conversion

//这⼀题没有什么算法思想，考察的是对程序控制的能⼒。⽤ 2 个变量保存⽅向，当垂直输出的⾏数达到了规定的⽬标⾏数以后，
//需要从下往上转折到第⼀⾏，循环中控制好⽅向ji
func convert(s string, numRows int) string {
	matrix, down, up := make([][]byte, numRows, numRows), 0, numRows-2
	for i := 0; i != len(s); {
		if down != numRows {
			matrix[down] = append(matrix[down], byte(s[i]))
			down++
			i++
		} else if up > 0 {
			matrix[up] = append(matrix[up], byte(s[i]))
			up--
			i++
		} else {
			up = numRows - 2
			down = 0
		}
	}
	//遍历 matrix 中的每一行，将其字符追加到 solution 字节切片中，最终返回拼接后的字符串
	solution := make([]byte, 0, len(s))
	for _, row := range matrix {
		for _, item := range row {
			solution = append(solution, item)
		}
	}
	return string(solution)
}
