package xerrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrap_Nil(t *testing.T) {
	assert.Nil(t, Wrap(nil, "some context"))
}

func TestWrap_PreservesErrorChain(t *testing.T) {
	original := ErrNotFound
	wrapped := Wrap(original, "users.Service.GetUser")

	require.Error(t, wrapped)
	assert.Contains(t, wrapped.Error(), "users.Service.GetUser")
	assert.Contains(t, wrapped.Error(), "resource not found")

	// errors.Is should find the sentinel
	assert.True(t, errors.Is(wrapped, ErrNotFound))
	assert.False(t, errors.Is(wrapped, ErrUnauthorized))

	// errors.As should extract the DomainError
	var de *DomainError
	assert.True(t, errors.As(wrapped, &de))
	assert.Equal(t, CodeNotFound, Code(de.Code))
}

func TestWrapf_PreservesErrorChain(t *testing.T) {
	original := ErrInvalidInput
	wrapped := Wrapf(original, "users.Service.CreateUser: email=%s", "test@example.com")

	require.Error(t, wrapped)
	assert.Contains(t, wrapped.Error(), "users.Service.CreateUser: email=test@example.com")
	assert.Contains(t, wrapped.Error(), "invalid input")

	assert.True(t, errors.Is(wrapped, ErrInvalidInput))
}

func TestWrap_MultipleLayers(t *testing.T) {
	original := ErrNotFound
	layer1 := Wrap(original, "repository.GetUser")
	layer2 := Wrap(layer1, "service.GetUser")
	layer3 := Wrapf(layer2, "handler.GetUser: id=%s", "123")

	assert.True(t, errors.Is(layer3, ErrNotFound))

	var de *DomainError
	require.True(t, errors.As(layer3, &de))
	assert.Equal(t, CodeNotFound, Code(de.Code))
	// The full error chain message should contain all wrapped contexts
	fullMsg := layer3.Error()
	assert.Contains(t, fullMsg, "handler.GetUser")
	assert.Contains(t, fullMsg, "service.GetUser")
	assert.Contains(t, fullMsg, "repository.GetUser")
	// The DomainError's Message field is just the sentinel's message
	assert.Equal(t, "resource not found", de.Message)
}

func TestWrap_CustomError(t *testing.T) {
	custom := New("custom_code", 418, "custom message")
	wrapped := Wrap(custom, "added context")

	assert.True(t, errors.Is(wrapped, custom))

	var de *DomainError
	require.True(t, errors.As(wrapped, &de))
	assert.Equal(t, "custom_code", de.Code)
	assert.Equal(t, 418, de.Status)
}

func TestAs_NonDomainError(t *testing.T) {
	err := errors.New("plain error")
	var de *DomainError
	_, ok := As(err)
	assert.False(t, ok)
	assert.False(t, errors.As(err, &de))
}

func TestIs_Sentinel(t *testing.T) {
	assert.True(t, Is(ErrNotFound, ErrNotFound))
	assert.False(t, Is(ErrNotFound, ErrUnauthorized))
}

func TestCodeConstants(t *testing.T) {
	tests := []struct {
		code Code
		str  string
	}{
		{CodeNotFound, "not_found"},
		{CodeAlreadyExists, "already_exists"},
		{CodeInvalidInput, "invalid_input"},
		{CodeUnauthorized, "unauthorized"},
		{CodeForbidden, "forbidden"},
		{CodeConflict, "conflict"},
		{CodeUnprocessableEntity, "unprocessable_entity"},
		{CodeDeviceFingerprintMismatch, "device_fingerprint_mismatch"},
		{CodeRequestCanceled, "request_canceled"},
		{CodeTimeout, "timeout"},
		{CodeInternalError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			assert.Equal(t, tt.str, string(tt.code))
		})
	}
}

func TestSentinelCodesMatchConstants(t *testing.T) {
	assert.Equal(t, string(CodeNotFound), ErrNotFound.Code)
	assert.Equal(t, string(CodeAlreadyExists), ErrAlreadyExists.Code)
	assert.Equal(t, string(CodeInvalidInput), ErrInvalidInput.Code)
	assert.Equal(t, string(CodeUnauthorized), ErrUnauthorized.Code)
	assert.Equal(t, string(CodeForbidden), ErrForbidden.Code)
	assert.Equal(t, string(CodeConflict), ErrConflict.Code)
}

func TestWithFields(t *testing.T) {
	de := InvalidInput("validation failed")
	de = WithFields(de, map[string][]string{"email": {"invalid format"}})

	require.NotNil(t, de.Fields)
	assert.Equal(t, []string{"invalid format"}, de.Fields["email"])
}
