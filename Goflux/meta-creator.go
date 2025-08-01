package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MetaInfo 元数据信息结构
type MetaInfo struct {
	// 基本文件信息
	FileName   string    `json:"file_name"`
	FileSize   int64     `json:"file_size"`
	FileHash   string    `json:"file_hash"` // 整个文件的SHA-256哈希
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"` // 原文件的修改时间

	// 分片信息
	PieceSize   int64    `json:"piece_size"`   // 分片大小（字节）
	PieceCount  int      `json:"piece_count"`  // 分片总数
	PieceHashes []string `json:"piece_hashes"` // 每个分片的SHA-256哈希列表

	// 扩展信息
	Version   string `json:"version"` // 元数据格式版本
	Comment   string `json:"comment,omitempty"`
	CreatedBy string `json:"created_by"` // 创建工具标识
}

// PieceInfo 分片信息
type PieceInfo struct {
	Index  int    `json:"index"`
	Hash   string `json:"hash"`
	Size   int64  `json:"size"`
	Offset int64  `json:"offset"`
}

const (
	MetaVersion  = "1.0"
	CreatedBy    = "meta-creator/1.0"
	BufferSize   = 64 * 1024 // 64KB 缓冲区
	ProgressStep = 10        // 每处理多少个分片显示一次进度
)

// 解析大小字符串（支持KB, MB, GB等单位）
func parseSizeString(sizeStr string) (int64, error) {
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))

	// 如果是纯数字，按字节处理
	if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		return size, nil
	}

	// 处理带单位的情况
	var multiplier int64 = 1
	var numStr string

	if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		numStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(sizeStr, "GB")
	} else if strings.HasSuffix(sizeStr, "B") {
		multiplier = 1
		numStr = strings.TrimSuffix(sizeStr, "B")
	} else {
		// 尝试解析K, M, G简写
		if strings.HasSuffix(sizeStr, "K") {
			multiplier = 1024
			numStr = strings.TrimSuffix(sizeStr, "K")
		} else if strings.HasSuffix(sizeStr, "M") {
			multiplier = 1024 * 1024
			numStr = strings.TrimSuffix(sizeStr, "M")
		} else if strings.HasSuffix(sizeStr, "G") {
			multiplier = 1024 * 1024 * 1024
			numStr = strings.TrimSuffix(sizeStr, "G")
		} else {
			return 0, fmt.Errorf("invalid size format: %s (支持格式: 256KB, 1MB, 2GB, 等)", sizeStr)
		}
	}

	size, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size number: %s", numStr)
	}

	return int64(size * float64(multiplier)), nil
}

// 格式化文件大小为可读字符串
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.2f %s", float64(size)/float64(div), units[exp])
}

// 计算文件的SHA-256哈希（使用缓冲读取）
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	buffer := make([]byte, BufferSize)

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			hasher.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// 生成元数据结构（不写文件）
func GenerateMetaInfo(filePath string, pieceSize int64, comment string) (*MetaInfo, error) {
	// 检查输入文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法访问文件 %s: %v", filePath, err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("%s 是一个目录，请指定文件路径", filePath)
	}

	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return nil, fmt.Errorf("文件 %s 为空", filePath)
	}

	fmt.Printf("正在处理文件: %s\n", filePath)
	fmt.Printf("文件大小: %s (%d 字节)\n", formatFileSize(fileSize), fileSize)
	fmt.Printf("分片大小: %s (%d 字节)\n", formatFileSize(pieceSize), pieceSize)

	// 计算文件总哈希
	fmt.Print("计算文件哈希...")
	fileHash, err := calculateFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("计算文件哈希失败: %v", err)
	}
	fmt.Println(" 完成")

	// 计算分片数量
	pieceCount := int((fileSize + pieceSize - 1) / pieceSize) // 向上取整
	fmt.Printf("将分割为 %d 个分片\n", pieceCount)

	// 计算每个分片的哈希
	pieceHashes, err := calculatePieceHashes(filePath, pieceSize, pieceCount)
	if err != nil {
		return nil, fmt.Errorf("计算分片哈希失败: %v", err)
	}

	// 创建元数据结构
	metaInfo := &MetaInfo{
		FileName:    filepath.Base(filePath),
		FileSize:    fileSize,
		FileHash:    fileHash,
		CreatedAt:   time.Now(),
		ModifiedAt:  fileInfo.ModTime(),
		PieceSize:   pieceSize,
		PieceCount:  pieceCount,
		PieceHashes: pieceHashes,
		Version:     MetaVersion,
		Comment:     comment,
		CreatedBy:   CreatedBy,
	}

	return metaInfo, nil
}

// 计算分片哈希（使用缓冲读取）
func calculatePieceHashes(filePath string, pieceSize int64, pieceCount int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, BufferSize)
	var pieceHashes []string
	buffer := make([]byte, pieceSize)

	fmt.Print("计算分片哈希: ")
	for i := 0; i < pieceCount; i++ {
		// 读取分片数据
		var totalRead int64
		hasher := sha256.New()

		for totalRead < pieceSize {
			remaining := pieceSize - totalRead
			if remaining > int64(len(buffer)) {
				remaining = int64(len(buffer))
			}

			n, err := reader.Read(buffer[:remaining])
			if n > 0 {
				hasher.Write(buffer[:n])
				totalRead += int64(n)
			}

			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("读取分片 %d 失败: %v", i, err)
			}
		}

		pieceHash := hex.EncodeToString(hasher.Sum(nil))
		pieceHashes = append(pieceHashes, pieceHash)

		// 显示进度
		if (i+1)%ProgressStep == 0 || i == pieceCount-1 {
			fmt.Printf("%d/%d ", i+1, pieceCount)
		}
	}
	fmt.Println("完成")

	return pieceHashes, nil
}

// 写入元数据文件
func WriteMetaFile(metaInfo *MetaInfo, outputPath string) error {
	// 确保输出目录存在
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 将元数据序列化为JSON
	jsonData, err := json.MarshalIndent(metaInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("写入元数据文件失败: %v", err)
	}

	return nil
}

// 检查输出文件是否已存在
func checkOutputFileExists(outputPath string, force bool) error {
	if _, err := os.Stat(outputPath); err == nil {
		if !force {
			return fmt.Errorf("输出文件已存在: %s (使用 -force 参数覆盖)", outputPath)
		}
		fmt.Printf("警告: 覆盖已存在的文件: %s\n", outputPath)
	}
	return nil
}

// 创建元数据文件（完整流程）
func CreateMetaFile(filePath string, pieceSize int64, outputPath string, comment string, force bool) (*MetaInfo, error) {
	// 生成输出文件路径
	if outputPath == "" {
		outputPath = filePath + ".meta"
	}

	// 检查输出文件是否已存在
	if err := checkOutputFileExists(outputPath, force); err != nil {
		return nil, err
	}

	// 生成元数据结构
	metaInfo, err := GenerateMetaInfo(filePath, pieceSize, comment)
	if err != nil {
		return nil, err
	}

	// 写入文件
	if err := WriteMetaFile(metaInfo, outputPath); err != nil {
		return nil, err
	}

	fmt.Printf("元数据文件已创建: %s\n", outputPath)
	fmt.Printf("文件哈希: %s\n", metaInfo.FileHash[:16]+"...")

	return metaInfo, nil
}

// ParseMetaFile 解析元数据文件
func ParseMetaFile(filePath string) (*MetaInfo, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("元数据文件不存在: %s", filePath)
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取元数据文件失败: %v", err)
	}

	// 解析JSON
	var metaInfo MetaInfo
	if err := json.Unmarshal(data, &metaInfo); err != nil {
		return nil, fmt.Errorf("解析元数据文件失败: %v", err)
	}

	// 验证元数据格式
	if err := validateMetaInfo(&metaInfo); err != nil {
		return nil, err
	}

	return &metaInfo, nil
}

// 验证元数据信息的完整性
func validateMetaInfo(metaInfo *MetaInfo) error {
	if metaInfo.FileName == "" {
		return fmt.Errorf("元数据文件格式错误: 缺少文件名")
	}

	if metaInfo.FileSize <= 0 {
		return fmt.Errorf("元数据文件格式错误: 文件大小无效")
	}

	if metaInfo.PieceSize <= 0 {
		return fmt.Errorf("元数据文件格式错误: 分片大小无效")
	}

	if len(metaInfo.PieceHashes) != metaInfo.PieceCount {
		return fmt.Errorf("元数据文件格式错误: 分片哈希数量不匹配")
	}

	// 验证哈希格式
	for i, hash := range metaInfo.PieceHashes {
		if len(hash) != 64 { // SHA-256 哈希应该是64个字符
			return fmt.Errorf("元数据文件格式错误: 分片 %d 哈希格式无效", i)
		}

		// 验证是否为有效的十六进制字符串
		if _, err := hex.DecodeString(hash); err != nil {
			return fmt.Errorf("元数据文件格式错误: 分片 %d 哈希不是有效的十六进制: %v", i, err)
		}
	}

	return nil
}

// 验证元数据文件的完整性
func VerifyMetaFile(metaPath, originalFilePath string) error {
	metaInfo, err := ParseMetaFile(metaPath)
	if err != nil {
		return fmt.Errorf("解析元数据文件失败: %v", err)
	}

	// 检查原文件是否存在
	fileInfo, err := os.Stat(originalFilePath)
	if err != nil {
		return fmt.Errorf("无法访问原文件: %v", err)
	}

	// 验证文件大小
	if fileInfo.Size() != metaInfo.FileSize {
		return fmt.Errorf("文件大小不匹配: 期望 %d，实际 %d", metaInfo.FileSize, fileInfo.Size())
	}

	// 验证文件哈希
	fmt.Print("验证文件哈希...")
	fileHash, err := calculateFileHash(originalFilePath)
	if err != nil {
		return fmt.Errorf("计算文件哈希失败: %v", err)
	}

	if fileHash != metaInfo.FileHash {
		return fmt.Errorf("文件哈希不匹配")
	}
	fmt.Println(" 通过")

	// 验证分片哈希
	if err := verifyPieceHashes(originalFilePath, metaInfo); err != nil {
		return err
	}

	return nil
}

// 验证分片哈希
func verifyPieceHashes(filePath string, metaInfo *MetaInfo) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, BufferSize)
	buffer := make([]byte, metaInfo.PieceSize)

	fmt.Print("验证分片哈希: ")

	for i := 0; i < metaInfo.PieceCount; i++ {
		var totalRead int64
		hasher := sha256.New()

		for totalRead < metaInfo.PieceSize {
			remaining := metaInfo.PieceSize - totalRead
			if remaining > int64(len(buffer)) {
				remaining = int64(len(buffer))
			}

			n, err := reader.Read(buffer[:remaining])
			if n > 0 {
				hasher.Write(buffer[:n])
				totalRead += int64(n)
			}

			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("读取分片 %d 失败: %v", i, err)
			}
		}

		pieceHash := hex.EncodeToString(hasher.Sum(nil))
		if pieceHash != metaInfo.PieceHashes[i] {
			return fmt.Errorf("分片 %d 哈希不匹配", i)
		}

		if (i+1)%ProgressStep == 0 || i == metaInfo.PieceCount-1 {
			fmt.Printf("%d/%d ", i+1, metaInfo.PieceCount)
		}
	}
	fmt.Println("通过")

	return nil
}

// 显示元数据文件信息
func ShowMetaInfo(metaPath string) error {
	metaInfo, err := ParseMetaFile(metaPath)
	if err != nil {
		return err
	}

	fmt.Printf("=== 元数据文件信息 ===\n")
	fmt.Printf("文件名: %s\n", metaInfo.FileName)
	fmt.Printf("文件大小: %s (%d 字节)\n", formatFileSize(metaInfo.FileSize), metaInfo.FileSize)
	fmt.Printf("文件哈希: %s\n", metaInfo.FileHash)
	fmt.Printf("分片大小: %s (%d 字节)\n", formatFileSize(metaInfo.PieceSize), metaInfo.PieceSize)
	fmt.Printf("分片数量: %d\n", metaInfo.PieceCount)
	fmt.Printf("创建时间: %s\n", metaInfo.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("修改时间: %s\n", metaInfo.ModifiedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("格式版本: %s\n", metaInfo.Version)
	fmt.Printf("创建工具: %s\n", metaInfo.CreatedBy)

	if metaInfo.Comment != "" {
		fmt.Printf("备注: %s\n", metaInfo.Comment)
	}

	// 显示前几个分片的哈希作为示例
	fmt.Printf("\n分片哈希预览 (前5个):\n")
	for i := 0; i < len(metaInfo.PieceHashes) && i < 5; i++ {
		fmt.Printf("  分片 %d: %s\n", i, metaInfo.PieceHashes[i][:16]+"...")
	}

	if len(metaInfo.PieceHashes) > 5 {
		fmt.Printf("  ... 还有 %d 个分片\n", len(metaInfo.PieceHashes)-5)
	}

	return nil
}

// 显示帮助信息
func showHelp() {
	fmt.Printf("Meta Creator - 文件元数据生成工具\n\n")
	fmt.Printf("用法:\n")
	fmt.Printf("  创建元数据文件:\n")
	fmt.Printf("    %s -file <文件路径> [-piece-size <分片大小>] [-output <输出路径>] [-comment <备注>] [-force]\n\n", os.Args[0])
	fmt.Printf("  验证元数据文件:\n")
	fmt.Printf("    %s -verify <元数据文件路径> -file <原文件路径>\n\n", os.Args[0])
	fmt.Printf("  显示元数据信息:\n")
	fmt.Printf("    %s -show <元数据文件路径>\n\n", os.Args[0])
	fmt.Printf("参数说明:\n")
	flag.PrintDefaults()
	fmt.Printf("\n分片大小支持的格式:\n")
	fmt.Printf("  256KB, 1MB, 2GB (支持B, KB, MB, GB后缀)\n")
	fmt.Printf("  也支持简写: 256K, 1M, 2G\n")
	fmt.Printf("  纯数字表示字节数: 262144\n")
	fmt.Printf("\n示例:\n")
	fmt.Printf("  %s -file movie.mp4 -piece-size 1MB\n", os.Args[0])
	fmt.Printf("  %s -file document.pdf -piece-size 512KB -output /tmp/document.meta -force\n", os.Args[0])
	fmt.Printf("  %s -verify movie.mp4.meta -file movie.mp4\n", os.Args[0])
	fmt.Printf("  %s -show document.meta\n", os.Args[0])
}

// 执行创建模式
func runCreateMode(filePath, pieceSizeStr, output, comment string, force bool) {
	// 解析分片大小
	pSize, err := parseSizeString(pieceSizeStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}

	if pSize <= 0 {
		fmt.Fprintln(os.Stderr, "错误: 分片大小必须大于0")
		os.Exit(1)
	}

	// 创建元数据文件
	metaInfo, err := CreateMetaFile(filePath, pSize, output, comment, force)
	if err != nil {
		fmt.Fprintln(os.Stderr, "创建元数据文件失败:", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ 成功创建元数据文件！\n")
	fmt.Printf("文件信息: %s (%s)\n", metaInfo.FileName, formatFileSize(metaInfo.FileSize))
	fmt.Printf("分片信息: %d 个分片，每片 %s\n", metaInfo.PieceCount, formatFileSize(metaInfo.PieceSize))
}

// 执行验证模式
func runVerifyMode(verifyPath, filePath string) {
	fmt.Printf("验证元数据文件: %s\n", verifyPath)
	fmt.Printf("对应原文件: %s\n", filePath)

	if err := VerifyMetaFile(verifyPath, filePath); err != nil {
		fmt.Fprintln(os.Stderr, "验证失败:", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 验证通过！元数据文件与原文件完全一致。\n")
}

// 执行显示模式
func runShowMode(showPath string) {
	if err := ShowMetaInfo(showPath); err != nil {
		fmt.Fprintln(os.Stderr, "显示元数据信息失败:", err)
		os.Exit(1)
	}
}

func main() {
	var (
		filePath  = flag.String("file", "", "要处理的文件路径 (必需)")
		pieceSize = flag.String("piece-size", "256KB", "分片大小 (如: 256KB, 1MB, 2GB)")
		output    = flag.String("output", "", "输出的.meta文件路径 (默认: 原文件名.meta)")
		comment   = flag.String("comment", "", "元数据文件的备注信息")
		force     = flag.Bool("force", false, "强制覆盖已存在的输出文件")
		verify    = flag.String("verify", "", "验证模式: 验证元数据文件与原文件的一致性")
		show      = flag.String("show", "", "显示模式: 显示元数据文件的详细信息")
		help      = flag.Bool("help", false, "显示帮助信息")
	)

	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// 验证模式
	if *verify != "" {
		if *filePath == "" {
			fmt.Fprintln(os.Stderr, "错误: 验证模式需要同时指定 -verify 和 -file 参数")
			os.Exit(1)
		}
		runVerifyMode(*verify, *filePath)
		return
	}

	// 显示模式
	if *show != "" {
		runShowMode(*show)
		return
	}

	// 创建模式 - 检查必需参数
	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须指定 -file 参数")
		fmt.Fprintln(os.Stderr, "使用 -help 查看详细帮助")
		os.Exit(1)
	}

	// 执行创建模式
	runCreateMode(*filePath, *pieceSize, *output, *comment, *force)
}
