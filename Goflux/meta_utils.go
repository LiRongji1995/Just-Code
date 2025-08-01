package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MetaUtils 元数据工具结构
type MetaUtils struct{}

// NewMetaUtils 创建元数据工具实例
func NewMetaUtils() *MetaUtils {
	return &MetaUtils{}
}

// BatchCreateMeta 批量创建元数据文件
func (mu *MetaUtils) BatchCreateMeta(directory string, pieceSize int64, extensions []string, force bool) error {
	fmt.Printf("开始批量处理目录: %s\n", directory)

	var processedFiles []string
	var failedFiles []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 检查文件扩展名
		if len(extensions) > 0 {
			ext := strings.ToLower(filepath.Ext(path))
			found := false
			for _, allowedExt := range extensions {
				if ext == strings.ToLower(allowedExt) {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		// 跳过已存在的.meta文件
		if strings.HasSuffix(strings.ToLower(path), ".meta") {
			return nil
		}

		fmt.Printf("处理文件: %s\n", path)

		// 生成元数据
		_, err = CreateMetaFile(path, pieceSize, "", "", force)
		if err != nil {
			fmt.Printf("  失败: %v\n", err)
			failedFiles = append(failedFiles, path)
		} else {
			fmt.Printf("  成功\n")
			processedFiles = append(processedFiles, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("遍历目录失败: %v", err)
	}

	// 显示统计信息
	fmt.Printf("\n=== 批量处理完成 ===\n")
	fmt.Printf("成功处理: %d 个文件\n", len(processedFiles))
	fmt.Printf("处理失败: %d 个文件\n", len(failedFiles))

	if len(failedFiles) > 0 {
		fmt.Printf("\n失败的文件:\n")
		for _, file := range failedFiles {
			fmt.Printf("  %s\n", file)
		}
	}

	return nil
}

// CompareMeta 比较两个元数据文件的差异
func (mu *MetaUtils) CompareMeta(metaPath1, metaPath2 string) error {
	meta1, err := ParseMetaFile(metaPath1)
	if err != nil {
		return fmt.Errorf("解析第一个元数据文件失败: %v", err)
	}

	meta2, err := ParseMetaFile(metaPath2)
	if err != nil {
		return fmt.Errorf("解析第二个元数据文件失败: %v", err)
	}

	fmt.Printf("=== 元数据文件比较 ===\n")
	fmt.Printf("文件1: %s\n", metaPath1)
	fmt.Printf("文件2: %s\n", metaPath2)
	fmt.Printf("\n")

	// 比较基本信息
	different := false

	if meta1.FileName != meta2.FileName {
		fmt.Printf("文件名不同: %s vs %s\n", meta1.FileName, meta2.FileName)
		different = true
	}

	if meta1.FileSize != meta2.FileSize {
		fmt.Printf("文件大小不同: %d vs %d\n", meta1.FileSize, meta2.FileSize)
		different = true
	}

	if meta1.FileHash != meta2.FileHash {
		fmt.Printf("文件哈希不同: %s vs %s\n", meta1.FileHash[:16]+"...", meta2.FileHash[:16]+"...")
		different = true
	}

	if meta1.PieceSize != meta2.PieceSize {
		fmt.Printf("分片大小不同: %d vs %d\n", meta1.PieceSize, meta2.PieceSize)
		different = true
	}

	if meta1.PieceCount != meta2.PieceCount {
		fmt.Printf("分片数量不同: %d vs %d\n", meta1.PieceCount, meta2.PieceCount)
		different = true
	}

	// 比较分片哈希
	if len(meta1.PieceHashes) != len(meta2.PieceHashes) {
		fmt.Printf("分片哈希数量不同: %d vs %d\n", len(meta1.PieceHashes), len(meta2.PieceHashes))
		different = true
	} else {
		differentPieces := 0
		for i := 0; i < len(meta1.PieceHashes); i++ {
			if meta1.PieceHashes[i] != meta2.PieceHashes[i] {
				differentPieces++
			}
		}
		if differentPieces > 0 {
			fmt.Printf("不同的分片哈希: %d/%d\n", differentPieces, len(meta1.PieceHashes))
			different = true
		}
	}

	if !different {
		fmt.Printf("✓ 两个元数据文件完全相同\n")
	} else {
		fmt.Printf("✗ 两个元数据文件存在差异\n")
	}

	return nil
}

// GetPieceInfo 获取指定分片的详细信息
func (mu *MetaUtils) GetPieceInfo(metaPath string, pieceIndex int) (*PieceInfo, error) {
	metaInfo, err := ParseMetaFile(metaPath)
	if err != nil {
		return nil, err
	}

	if pieceIndex < 0 || pieceIndex >= metaInfo.PieceCount {
		return nil, fmt.Errorf("分片索引超出范围: %d (总共 %d 个分片)", pieceIndex, metaInfo.PieceCount)
	}

	// 计算分片的偏移量和大小
	offset := int64(pieceIndex) * metaInfo.PieceSize
	size := metaInfo.PieceSize

	// 最后一个分片可能较小
	if pieceIndex == metaInfo.PieceCount-1 {
		size = metaInfo.FileSize - offset
	}

	return &PieceInfo{
		Index:  pieceIndex,
		Hash:   metaInfo.PieceHashes[pieceIndex],
		Size:   size,
		Offset: offset,
	}, nil
}

// ListPieces 列出所有分片信息
func (mu *MetaUtils) ListPieces(metaPath string, limit int) error {
	metaInfo, err := ParseMetaFile(metaPath)
	if err != nil {
		return err
	}

	fmt.Printf("=== 分片列表 ===\n")
	fmt.Printf("文件: %s\n", metaInfo.FileName)
	fmt.Printf("总分片数: %d\n", metaInfo.PieceCount)
	fmt.Printf("\n")

	maxShow := metaInfo.PieceCount
	if limit > 0 && limit < maxShow {
		maxShow = limit
	}

	for i := 0; i < maxShow; i++ {
		pieceInfo, err := mu.GetPieceInfo(metaPath, i)
		if err != nil {
			return err
		}

		fmt.Printf("分片 %4d: 偏移 %12d, 大小 %8s, 哈希 %s\n",
			pieceInfo.Index,
			pieceInfo.Offset,
			formatFileSize(pieceInfo.Size),
			pieceInfo.Hash[:16]+"...")
	}

	if maxShow < metaInfo.PieceCount {
		fmt.Printf("... 还有 %d 个分片 (使用 -limit 0 显示全部)\n", metaInfo.PieceCount-maxShow)
	}

	return nil
}

// ConvertPieceSize 转换元数据文件的分片大小
func (mu *MetaUtils) ConvertPieceSize(metaPath, outputPath string, newPieceSize int64) error {
	// 解析原元数据
	metaInfo, err := ParseMetaFile(metaPath)
	if err != nil {
		return fmt.Errorf("解析原元数据文件失败: %v", err)
	}

	// 检查是否需要转换
	if metaInfo.PieceSize == newPieceSize {
		return fmt.Errorf("新分片大小与原分片大小相同，无需转换")
	}

	fmt.Printf("转换分片大小: %s -> %s\n",
		formatFileSize(metaInfo.PieceSize),
		formatFileSize(newPieceSize))

	// 找到原文件路径
	originalFile := strings.TrimSuffix(metaPath, ".meta")
	if _, err := os.Stat(originalFile); os.IsNotExist(err) {
		return fmt.Errorf("找不到原文件: %s", originalFile)
	}

	// 生成新的元数据
	newMetaInfo, err := GenerateMetaInfo(originalFile, newPieceSize, metaInfo.Comment)
	if err != nil {
		return fmt.Errorf("生成新元数据失败: %v", err)
	}

	// 保留一些原有信息
	newMetaInfo.Comment = fmt.Sprintf("从 %s 转换而来 (原分片大小: %s)",
		formatFileSize(metaInfo.PieceSize), formatFileSize(metaInfo.PieceSize))

	// 写入新文件
	if err := WriteMetaFile(newMetaInfo, outputPath); err != nil {
		return fmt.Errorf("写入新元数据文件失败: %v", err)
	}

	fmt.Printf("转换完成: %s\n", outputPath)
	fmt.Printf("原分片数: %d -> 新分片数: %d\n", metaInfo.PieceCount, newMetaInfo.PieceCount)

	return nil
}

// ValidateIntegrity 深度验证文件完整性
func (mu *MetaUtils) ValidateIntegrity(metaPath string, originalFile string, checkAll bool) error {
	metaInfo, err := ParseMetaFile(metaPath)
	if err != nil {
		return err
	}

	fmt.Printf("验证文件完整性: %s\n", originalFile)

	// 基本验证
	if err := VerifyMetaFile(metaPath, originalFile); err != nil {
		return err
	}

	if !checkAll {
		return nil
	}

	// 额外的深度检查
	fmt.Printf("执行深度完整性检查...\n")

	// 检查文件是否可读
	file, err := os.Open(originalFile)
	if err != nil {
		return fmt.Errorf("无法打开文件进行深度检查: %v", err)
	}
	defer file.Close()

	// 检查每个分片的边界
	buffer := make([]byte, 1024) // 小缓冲区用于边界检查
	for i := 0; i < metaInfo.PieceCount; i++ {
		offset := int64(i) * metaInfo.PieceSize

		// 定位到分片开始位置
		if _, err := file.Seek(offset, 0); err != nil {
			return fmt.Errorf("无法定位到分片 %d: %v", i, err)
		}

		// 尝试读取一小块数据
		n, err := file.Read(buffer)
		if err != nil && n == 0 {
			return fmt.Errorf("分片 %d 读取失败: %v", i, err)
		}
	}

	fmt.Printf("✓ 深度完整性检查通过\n")
	return nil
}

// ==== 构建和测试脚本 ====

// build.sh
const BuildScript = `#!/bin/bash
# 构建脚本

set -e

echo "开始构建 meta-creator..."

# 设置版本信息
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
GO_VERSION=$(go version | awk '{print $3}')

# 编译参数
LDFLAGS="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}' -X 'main.GoVersion=${GO_VERSION}'"

# 清理旧的构建文件
rm -f meta-creator

# 编译
echo "编译中..."
go build -ldflags "${LDFLAGS}" -o meta-creator .

# 检查编译结果
if [ -f meta-creator ]; then
    echo "✓ 编译成功"
    echo "版本: ${VERSION}"
    echo "构建时间: ${BUILD_TIME}"
    echo "Go版本: ${GO_VERSION}"
    
    # 显示文件信息
    ls -lh meta-creator
    
    echo ""
    echo "使用方法:"
    echo "  ./meta-creator -help"
else
    echo "✗ 编译失败"
    exit 1
fi
`

// test.sh
const TestScript = `#!/bin/bash
# 测试脚本

set -e

echo "开始测试 meta-creator..."

# 检查是否已编译
if [ ! -f meta-creator ]; then
    echo "请先运行 ./build.sh 编译程序"
    exit 1
fi

# 创建测试目录
TEST_DIR="test_files"
mkdir -p ${TEST_DIR}

# 生成测试文件
echo "创建测试文件..."

# 小文件测试
echo "Hello, World! This is a test file for meta-creator." > ${TEST_DIR}/small.txt

# 中等文件测试 (1MB)
dd if=/dev/urandom of=${TEST_DIR}/medium.bin bs=1M count=1 2>/dev/null

# 大文件测试 (如果有足够空间的话)
if [ "${LARGE_TEST:-0}" = "1" ]; then
    echo "创建大文件测试..."
    dd if=/dev/urandom of=${TEST_DIR}/large.bin bs=1M count=100 2>/dev/null
fi

echo ""
echo "=== 测试1: 基本功能 ==="
./meta-creator -file ${TEST_DIR}/small.txt -piece-size 16

echo ""
echo "=== 测试2: 不同分片大小 ==="
./meta-creator -file ${TEST_DIR}/medium.bin -piece-size 256KB -force

echo ""
echo "=== 测试3: 显示元数据信息 ==="
./meta-creator -show ${TEST_DIR}/small.txt.meta

echo ""
echo "=== 测试4: 验证完整性 ==="
./meta-creator -verify ${TEST_DIR}/small.txt.meta -file ${TEST_DIR}/small.txt

echo ""
echo "=== 测试5: 错误处理 ==="
# 测试不存在的文件
./meta-creator -file non_existent_file.txt 2>/dev/null && echo "应该失败但没有失败" || echo "✓ 正确处理不存在的文件"

# 测试无效的分片大小
./meta-creator -file ${TEST_DIR}/small.txt -piece-size "invalid" 2>/dev/null && echo "应该失败但没有失败" || echo "✓ 正确处理无效分片大小"

echo ""
echo "=== 测试完成 ==="
echo "测试文件位于: ${TEST_DIR}/"
echo "可以手动检查生成的 .meta 文件"

# 清理选项
if [ "${CLEANUP:-0}" = "1" ]; then
    echo "清理测试文件..."
    rm -rf ${TEST_DIR}
fi
`

// Makefile
const Makefile = `# Makefile for meta-creator

.PHONY: build test clean install help

# 默认目标
all: build

# 编译
build:
	@echo "编译 meta-creator..."
	@chmod +x build.sh
	@./build.sh

# 测试
test: build
	@echo "运行测试..."
	@chmod +x test.sh
	@./test.sh

# 大文件测试
test-large: build
	@echo "运行大文件测试..."
	@chmod +x test.sh
	@LARGE_TEST=1 ./test.sh

# 测试并清理
test-clean: build
	@echo "运行测试并清理..."
	@chmod +x test.sh
	@CLEANUP=1 ./test.sh

# 安装到系统路径
install: build
	@echo "安装 meta-creator..."
	@sudo cp meta-creator /usr/local/bin/
	@echo "✓ 已安装到 /usr/local/bin/meta-creator"

# 清理
clean:
	@echo "清理构建文件..."
	@rm -f meta-creator
	@rm -rf test_files/
	@echo "✓ 清理完成"

# 帮助
help:
	@echo "可用的目标:"
	@echo "  build      - 编译程序"
	@echo "  test       - 运行基本测试"
	@echo "  test-large - 运行包含大文件的测试"
	@echo "  test-clean - 运行测试并清理"
	@echo "  install    - 安装到系统路径"
	@echo "  clean      - 清理构建文件"
	@echo "  help       - 显示此帮助信息"
`

func main() {
	// 这里可以添加一些工具函数的使用示例
	fmt.Println("Meta Creator 工具包")
	fmt.Println("包含以下组件:")
	fmt.Println("1. meta-creator - 主程序")
	fmt.Println("2. MetaUtils - 工具包")
	fmt.Println("3. 构建和测试脚本")

	// 示例：使用工具包
	utils := NewMetaUtils()

	// 如果有命令行参数，可以在这里处理工具包功能
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "batch":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "用法: meta-creator batch <目录> [分片大小] [扩展名...]")
				os.Exit(1)
			}

			directory := os.Args[2]
			pieceSize := int64(256 * 1024) // 默认256KB

			if len(os.Args) > 3 {
				if size, err := parseSizeString(os.Args[3]); err == nil {
					pieceSize = size
				}
			}

			var extensions []string
			if len(os.Args) > 4 {
				extensions = os.Args[4:]
			}

			if err := utils.BatchCreateMeta(directory, pieceSize, extensions, false); err != nil {
				fmt.Fprintln(os.Stderr, "批量处理失败:", err)
				os.Exit(1)
			}

		case "compare":
			if len(os.Args) != 4 {
				fmt.Fprintln(os.Stderr, "用法: meta-creator compare <meta1> <meta2>")
				os.Exit(1)
			}

			if err := utils.CompareMeta(os.Args[2], os.Args[3]); err != nil {
				fmt.Fprintln(os.Stderr, "比较失败:", err)
				os.Exit(1)
			}

		case "list":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "用法: meta-creator list <meta文件> [限制数量]")
				os.Exit(1)
			}

			limit := 20 // 默认显示20个
			if len(os.Args) > 3 {
				if l, err := strconv.Atoi(os.Args[3]); err == nil {
					limit = l
				}
			}

			if err := utils.ListPieces(os.Args[2], limit); err != nil {
				fmt.Fprintln(os.Stderr, "列出分片失败:", err)
				os.Exit(1)
			}

		default:
			fmt.Printf("未知命令: %s\n", os.Args[1])
			fmt.Println("可用命令: batch, compare, list")
		}
	}
}
