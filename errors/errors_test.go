package errors

import "testing"

func TestErrorFormatting(t *testing.T) {
	err := Error{
		Message: "boom",
		Code:    42,
		errType: ConfigError,
	}
	expected := "configuration error : code 42 : boom"
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestNewError(t *testing.T) {
	err := NewError(7, "bad config", CircuitStateError)
	cbErr, ok := err.(Error)
	if !ok {
		t.Fatalf("expected Error type, got %T", err)
	}
	if cbErr.Code != 7 || cbErr.Message != "bad config" || cbErr.errType != CircuitStateError {
		t.Fatalf("unexpected error fields: %#v", cbErr)
	}
}
