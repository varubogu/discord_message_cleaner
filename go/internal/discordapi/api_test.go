package discordapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/disgoorg/disgo/rest"
)

func TestClassify(t *testing.T) {
	e := func(status int, code rest.JSONErrorCode) error {
		return fmt.Errorf("wrapped: %w", &rest.Error{Code: code, Response: &http.Response{StatusCode: status}})
	}
	tests := []struct {
		err  error
		want ErrorKind
	}{
		{context.Canceled, KindCanceled},
		{errors.New("connection reset"), KindTemporary},
		{e(404, rest.JSONErrorCodeUnknownMessage), KindMessageGone},
		{e(404, rest.JSONErrorCodeUnknownChannel), KindChannelGone},
		{e(403, rest.JSONErrorCodeMissingAccess), KindChannelDenied},
		{e(403, rest.JSONErrorCodeLackPermissionsToPerformAction), KindChannelDenied},
		{e(400, rest.JSONErrorCodeOperationOnArchivedThread), KindChannelDenied},
		{e(400, rest.JSONErrorCodeMessageTooOldToBulkDelete), KindBadBulk},
		{e(403, rest.JSONErrorCodeCannotExecuteActionOnSystemMessage), KindUndeletable},
		{e(429, 0), KindTemporary},
		{e(502, 0), KindTemporary},
		{e(403, 0), KindChannelDenied},
	}
	for _, tt := range tests {
		if got := Classify(tt.err); got != tt.want {
			t.Errorf("Classify(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}
