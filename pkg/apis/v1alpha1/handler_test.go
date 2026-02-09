package v1alpha1

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler_NewError(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		expectedCode     int
		expectedMessage  string
		expectedRespCode int32
	}{
		{
			name: "ErrorWithCodeを渡した場合、指定されたコードとメッセージが返ること",
			err: &ErrorWithCode{
				Code:    http.StatusBadRequest,
				Message: "bad request error",
			},
			expectedCode:     http.StatusBadRequest,
			expectedMessage:  "bad request error",
			expectedRespCode: int32(http.StatusBadRequest),
		},
		{
			name:             "通常のerrorを渡した場合、500とエラーメッセージが返ること",
			err:              errors.New("unexpected error"),
			expectedCode:     http.StatusInternalServerError,
			expectedMessage:  "unexpected error",
			expectedRespCode: ErrorCodeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := &Handler{}
			ret := handler.NewError(t.Context(), tt.err)

			assert.NotNil(t, ret)
			assert.Equal(t, tt.expectedCode, ret.StatusCode)
			assert.Equal(t, tt.expectedRespCode, ret.Response.Code)
			assert.Equal(t, tt.expectedMessage, ret.Response.Message)
		})
	}
}
