package kiro

import (
	"github.com/bytedance/sonic"
)

// 高性能 JSON 配置 (匹配 kiro2api utils/json.go)
var (
	// FastestConfig 最快的 JSON 配置，用于性能关键路径
	FastestConfig = sonic.ConfigFastest

	// SafeConfig 安全的 JSON 配置，带有更多验证
	SafeConfig = sonic.ConfigStd
)

// FastMarshal 高性能 JSON 序列化
func FastMarshal(v any) ([]byte, error) {
	return FastestConfig.Marshal(v)
}

// FastUnmarshal 高性能 JSON 反序列化
func FastUnmarshal(data []byte, v any) error {
	return FastestConfig.Unmarshal(data, v)
}

// SafeMarshal 安全 JSON 序列化（带验证）
func SafeMarshal(v any) ([]byte, error) {
	return SafeConfig.Marshal(v)
}

// SafeUnmarshal 安全 JSON 反序列化（带验证）
func SafeUnmarshal(data []byte, v any) error {
	return SafeConfig.Unmarshal(data, v)
}

// MarshalIndent 带缩进的 JSON 序列化
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return SafeConfig.MarshalIndent(v, prefix, indent)
}

// jsonUnmarshal 内部使用的反序列化函数
func jsonUnmarshal(data []byte, v any) error {
	return FastestConfig.Unmarshal(data, v)
}
