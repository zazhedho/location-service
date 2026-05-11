package response

import (
	"net/http"

	"github.com/google/uuid"

	"location-service/pkg/messages"
)

type Errors struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type APIResponse struct {
	ID      uuid.UUID `json:"log_id"`
	Code    int       `json:"code,omitempty"`
	Status  bool      `json:"status"`
	Message string    `json:"message"`
	Data    any       `json:"data,omitempty"`
	Error   any       `json:"error,omitempty"`
}

func Response(code int, msg string, logID uuid.UUID, data any) *APIResponse {
	res := &APIResponse{
		ID:     logID,
		Code:   code,
		Data:   data,
		Status: code >= http.StatusOK && code < http.StatusMultipleChoices,
	}
	if res.Status {
		res.Message = msg
		return res
	}
	if msg == "" {
		msg = http.StatusText(code)
	}
	res.Message = errorTitle(code)
	res.Error = Errors{Code: code, Message: msg}
	return res
}

func ErrorResponse(code int, msg string, logID uuid.UUID, publicError string) *APIResponse {
	res := Response(code, msg, logID, nil)
	res.Error = Errors{Code: code, Message: publicError}
	return res
}

func InternalServerError(logID uuid.UUID) *APIResponse {
	res := ErrorResponse(http.StatusInternalServerError, messages.MsgSomethingWrong, logID, messages.MsgInternal)
	res.Message = messages.MsgSomethingWrong
	return res
}

func errorTitle(code int) string {
	if title := http.StatusText(code); title != "" {
		return title
	}
	return messages.MsgSomethingWrong
}
