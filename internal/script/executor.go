package script

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/dop251/goja"
)

// ScriptExecutor JavaScript 스크립트 실행기
type ScriptExecutor struct {
	vm   *goja.Runtime
	vars map[string]interface{}
}

// NewScriptExecutor 새로운 스크립트 실행기 생성
func NewScriptExecutor() *ScriptExecutor {
	vm := goja.New()
	executor := &ScriptExecutor{
		vm:   vm,
		vars: make(map[string]interface{}),
	}

	// 환경변수 객체 설정
	envObj := vm.NewObject()
	envObj.Set("set", executor.envSet)
	envObj.Set("get", executor.envGet)
	vm.Set("env", envObj)

	// crypto 함수들 설정
	vm.Set("sha256", executor.sha256Func)
	vm.Set("sha512", executor.sha512Func)

	// Date.now() 함수 (JavaScript 표준)
	vm.RunString(`
		if (typeof Date === 'undefined') {
			Date = {};
		}
		Date.now = function() {
			return new Date().getTime();
		};
	`)

	return executor
}

// envSet 환경변수 설정
func (e *ScriptExecutor) envSet(key string, value interface{}) {
	e.vars[key] = value
}

// envGet 환경변수 조회
func (e *ScriptExecutor) envGet(key string) interface{} {
	return e.vars[key]
}

// sha256Func SHA256 해시 함수
func (e *ScriptExecutor) sha256Func(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// sha512Func SHA512 해시 함수
func (e *ScriptExecutor) sha512Func(data string) string {
	hash := sha512.Sum512([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Execute 스크립트 실행
func (e *ScriptExecutor) Execute(script string) error {
	if script == "" {
		return nil
	}

	_, err := e.vm.RunString(script)
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	return nil
}

// GetVars 모든 환경변수 반환
func (e *ScriptExecutor) GetVars() map[string]interface{} {
	return e.vars
}

// GetVar 특정 환경변수 반환
func (e *ScriptExecutor) GetVar(key string) (interface{}, bool) {
	val, ok := e.vars[key]
	return val, ok
}
