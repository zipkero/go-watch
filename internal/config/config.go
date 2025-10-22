package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config yaml 구조
type Config struct {
	URL         string `yaml:"url"`
	Method      string `yaml:"method"`
	Requests    int    `yaml:"requests"`
	Concurrency int    `yaml:"concurrency"`
	Delay       int    `yaml:"delay"`

	// 출력 설정
	OutputFile       string `yaml:"output_file"`
	SaveResponseBody bool   `yaml:"save_response_body"`
}

// LoadConfig YAML 파일을 읽어서 Config 구조체로 반환
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	if config.Method == "" {
		config.Method = "GET"
	}

	// 유효성 검사
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate 설정값 유효성 검사
func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required")
	}

	if c.Requests <= 0 {
		return fmt.Errorf("requests must be greater than 0")
	}

	if c.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be greater than 0")
	}

	if c.Delay < 0 {
		return fmt.Errorf("delay must be greater than or equal to 0")
	}

	return nil
}
