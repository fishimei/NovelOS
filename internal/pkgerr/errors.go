// Package pkgerr 定义了应用程序的错误类型和错误处理工具。
// 提供结构化的错误信息，包含错误码、HTTP 状态码和错误消息。
package pkgerr

import (
	"errors"
	"net/http"
)

// Code 是错误码的类型定义。
type Code string

// 预定义的错误码常量。
const (
	CodeValidationError     Code = "VALIDATION_ERROR"     // 验证错误
	CodeNotFound            Code = "NOT_FOUND"            // 资源未找到
	CodeConflict            Code = "CONFLICT"             // 资源冲突
	CodeGenerationFailed    Code = "GENERATION_FAILED"    // AI 生成失败
	CodeConstraintViolation Code = "CONSTRAINT_VIOLATION" // 约束违反
	CodeInternalError       Code = "INTERNAL_ERROR"       // 内部错误
	CodeRunNotReady         Code = "RUN_NOT_READY"        // 运行未就绪
)

// Error 是应用程序错误结构体。
type Error struct {
	Code    Code   // 错误码
	Message string // 错误消息
	Status  int    // HTTP 状态码
	Cause   error  // 原始错误
}

// Error 实现 error 接口。
func (e *Error) Error() string {
	return e.Message
}

// Unwrap 实现 errors.Unwrap 接口，用于错误链追溯。
func (e *Error) Unwrap() error {
	return e.Cause
}

// New 创建新的应用程序错误。
func New(code Code, status int, message string, cause error) *Error {
	return &Error{Code: code, Status: status, Message: message, Cause: cause}
}

// Validation 创建验证错误。
func Validation(message string) *Error {
	return New(CodeValidationError, http.StatusBadRequest, message, nil)
}

// NotFound 创建未找到错误。
func NotFound(message string) *Error {
	return New(CodeNotFound, http.StatusNotFound, message, nil)
}

// Conflict 创建冲突错误。
func Conflict(code Code, message string) *Error {
	if code == "" {
		code = CodeConflict
	}
	return New(code, http.StatusConflict, message, nil)
}

// Internal 创建内部错误。
func Internal(message string, cause error) *Error {
	if message == "" {
		message = "internal error"
	}
	return New(CodeInternalError, http.StatusInternalServerError, message, cause)
}

// AsError 将任意错误转换为应用程序错误。
// 如果已经是应用程序错误则直接返回，否则包装为内部错误。
func AsError(err error) *Error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}

	return Internal("internal error", err)
}
