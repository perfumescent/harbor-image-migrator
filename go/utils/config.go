package utils

import (
    "fmt"
    "github.com/spf13/viper"
)

// InitConfig 初始化配置
func InitConfig(configPath string) (*viper.Viper, error) {
    v := viper.New()
    
    // 设置配置文件路径
    v.SetConfigFile(configPath)
    
    // 读取配置文件
    if err := v.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("读取配置文件失败: %w", err)
    }
    
    return v, nil
} 