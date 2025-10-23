package script

import (
	"fmt"
	"regexp"
	"strings"
)

var templateRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ReplaceTemplates 템플릿 변수를 실제 값으로 치환
func ReplaceTemplates(text string, vars map[string]interface{}) string {
	return templateRegex.ReplaceAllStringFunc(text, func(match string) string {
		// {{ 와 }} 제거
		varName := strings.TrimSpace(match[2 : len(match)-2])

		if val, ok := vars[varName]; ok {
			return fmt.Sprintf("%v", val)
		}

		// 변수가 없으면 원래 그대로 반환
		return match
	})
}

// ReplaceTemplatesInMap map의 모든 값에 대해 템플릿 치환
func ReplaceTemplatesInMap(m map[string]string, vars map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range m {
		result[key] = ReplaceTemplates(value, vars)
	}
	return result
}
