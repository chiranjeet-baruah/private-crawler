package composable_error

import (
	"fmt"
)

type ComposableError struct {
	code    string
	message string
}

func (ce ComposableError) Error() string {
	return fmt.Sprintf("[%s] %s", ce.code, ce.message)
}

func GetCode(err error) string {
	ce, ok := err.(ComposableError)
	if !ok {
		return "DEFAULT"
	}
	return ce.code
}

func ComposeWith(err error, code string, message string) error {
	ce, ok := err.(ComposableError)
	if !ok {
		return err
	}
	if code != "" {
		ce.code = code + "_" + ce.code
	}
	if message != "" {
		ce.message = message + ", " + ce.message
	}
	return ce
}

func New(code string, message string) ComposableError {
	return ComposableError{
		code:    code,
		message: message,
	}
}
