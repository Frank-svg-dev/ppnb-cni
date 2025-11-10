package utils

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GetRandomNumber(max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max)
}

func FirstIPFromCIDRStr(cidr string) string {
	// 截取掉掩码部分
	parts := strings.Split(cidr, "/")
	ipPart := parts[0]

	// 替换掉最后一段为 .1
	idx := strings.LastIndex(ipPart, ".")
	if idx == -1 {
		return ""
	}
	return ipPart[:idx+1] + "2"
}

func RandomPickAndRemove(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("读取目录失败: %w", err)
	}

	// 过滤只保留文件
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}

	if len(files) == 0 {
		return "", fmt.Errorf("目录中没有可用文件: %s", dir)
	}

	// 随机选择一个
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(files))
	chosen := files[randomIndex]
	fullPath := filepath.Join(dir, chosen)

	// 删除文件
	if err := os.Remove(fullPath); err != nil {
		return "", fmt.Errorf("删除文件失败 (%s): %w", chosen, err)
	}

	return chosen, nil
}
