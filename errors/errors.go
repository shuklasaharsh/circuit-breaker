package errors

import "fmt"

type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	errType ErrorFmt
}

func (e Error) Error() string {
	return fmt.Sprintf("%s : code %d : %s", e.errType, e.Code, e.Message)
}

func NewError(code int, message string, fmt ErrorFmt) error {
	return Error{
		Code:    code,
		Message: message,
		errType: fmt,
	}
}
