package httpx

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/gippuss/devmatch-back/internal/domain"
)

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteError(c *gin.Context, err error) {
	appErr := domain.AsAppError(err)
	_ = c.Error(err)
	c.AbortWithStatusJSON(appErr.Status, errorResponse{Error: errorBody{Code: appErr.Code, Message: appErr.Message}})
}

func WriteValidationError(c *gin.Context, message string) {
	c.AbortWithStatusJSON(domain.ErrValidation.Status, errorResponse{Error: errorBody{Code: domain.ErrValidation.Code, Message: message}})
}

func WriteBindError(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	if errors, ok := err.(validator.ValidationErrors); ok {
		ve = errors
		msgs := make([]string, 0, len(ve))
		for _, fe := range ve {
			msgs = append(msgs, fieldErrorMessage(fe))
		}
		WriteValidationError(c, strings.Join(msgs, "; "))
		return
	}
	WriteValidationError(c, "Invalid request body")
}

func fieldErrorMessage(fe validator.FieldError) string {
	field := strings.ToLower(fe.Field())
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fe.Param())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be %s or more", field, fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}
