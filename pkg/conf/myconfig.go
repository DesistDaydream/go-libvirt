package conf

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

// MyConfig 是 my_config 文件解析后的结果。
type MyConfig struct {
	IPs     []string
	URI     string
	Domains []string
}

func LoadMyConfig(path, name string) (*MyConfig, error) {
	v := viper.New()
	v.SetConfigName(name)
	v.AddConfigPath(path)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	return &MyConfig{
		IPs:     compact(v.GetStringSlice("ips")),
		URI:     v.GetString("uri"),
		Domains: compact(v.GetStringSlice("domain.domains")),
	}, nil
}

func compact(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
