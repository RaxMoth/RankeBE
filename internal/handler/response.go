package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error codes
const (
	CodeValidationError  = "VALIDATION_ERROR"
	CodeInvalidValueType = "INVALID_VALUE_TYPE"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeListNotFound     = "LIST_NOT_FOUND"
	CodeInternalError    = "INTERNAL_ERROR"
)

type successEnvelope struct {
	Data any `json:"data"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

func Success(c *gin.Context, status int, data any) {
	c.JSON(status, successEnvelope{Data: data})
}

func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, errorEnvelope{Error: errorBody{Code: code, Message: message}})
}

func ValidationError(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, CodeValidationError, message)
}

func Unauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, CodeUnauthorized, message)
}

func Forbidden(c *gin.Context) {
	Fail(c, http.StatusForbidden, CodeForbidden, "you do not have permission to perform this action")
}

func NotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, CodeNotFound, message)
}

func InternalError(c *gin.Context) {
	Fail(c, http.StatusInternalServerError, CodeInternalError, "internal server error")
}
