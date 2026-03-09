package errors

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	cause := errors.New("original error")
	appErr := &AppError{
		Technical: "technical message",
		Cause:     cause,
	}

	expected := "technical message: original error"
	if appErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, appErr.Error())
	}

	appErrNoCause := &AppError{
		Technical: "technical message",
	}
	expectedNoCause := "technical message"
	if appErrNoCause.Error() != expectedNoCause {
		t.Errorf("expected %q, got %q", expectedNoCause, appErrNoCause.Error())
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	appErr := &AppError{
		Cause: cause,
	}

	if appErr.Unwrap() != cause {
		t.Errorf("expected %v, got %v", cause, appErr.Unwrap())
	}
}

func TestAppError_I18nData(t *testing.T) {
	args := map[string]interface{}{"key": "value"}
	appErr := &AppError{
		I18nKey:  "test.key",
		I18nArgs: args,
	}

	key, gotArgs := appErr.I18nData()
	if key != "test.key" {
		t.Errorf("expected test.key, got %s", key)
	}
	if gotArgs["key"] != "value" {
		t.Errorf("expected value, got %v", gotArgs["key"])
	}
}

func TestNewAppError(t *testing.T) {
	cause := errors.New("cause")
	args := map[string]interface{}{"arg": 1}
	err := NewAppError("key", "tech", args, cause)

	if err.I18nKey != "key" || err.Technical != "tech" || err.Cause != cause || err.I18nArgs["arg"] != 1 {
		t.Errorf("NewAppError failed to initialize fields correctly")
	}
	if err.Code != "key" {
		t.Errorf("expected code key, got %s", err.Code)
	}
}

func TestErrorConstructors(t *testing.T) {
	cause := errors.New("io error")

	tests := []struct {
		name     string
		err      *AppError
		wantKey  string
		wantTech string
	}{
		{
			"FileNotFound",
			FileNotFoundError("path/to/file", cause),
			ErrCodeFileNotFound,
			"file not found: path/to/file",
		},
		{
			"FileAccess",
			FileAccessError("path/to/file", cause),
			ErrCodeFileAccess,
			"failed to access file: path/to/file",
		},
		{
			"Duplicate",
			DuplicateError("category", "rock"),
			ErrCodeDuplicate,
			"duplicate category: rock",
		},
		{
			"Database",
			DatabaseError("insert", cause),
			ErrCodeDatabase,
			"database insert failed",
		},
		{
			"Network",
			NetworkError("GET", cause),
			ErrCodeNetwork,
			"network error during GET",
		},
		{
			"WebDAV",
			WebDAVError("PROPFIND", cause),
			ErrCodeWebDAV,
			"WebDAV PROPFIND failed",
		},
		{
			"Metadata",
			MetadataError("file.mp3", cause),
			ErrCodeMetadata,
			"failed to parse metadata from file.mp3",
		},
		{
			"CoverDownload",
			CoverDownloadError("Artist", "Album", cause),
			ErrCodeCoverDownload,
			"failed to download cover for Artist - Album",
		},
		{
			"MusicBrainz",
			MusicBrainzError("Artist", cause),
			ErrCodeMusicBrainz,
			"MusicBrainz lookup failed for Artist",
		},
		{
			"InvalidInput",
			InvalidInputError("field1", "too long"),
			ErrCodeInvalidInput,
			"invalid input for field1: too long",
		},
		{
			"Permission",
			PermissionError("delete", cause),
			ErrCodePermission,
			"permission denied for delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.I18nKey != tt.wantKey {
				t.Errorf("got key %s, want %s", tt.err.I18nKey, tt.wantKey)
			}
			if tt.err.Technical != tt.wantTech {
				t.Errorf("got tech %s, want %s", tt.err.Technical, tt.wantTech)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	// Test wrapping a normal error
	cause := errors.New("standard error")
	wrapped := WrapError(cause, "custom.key", map[string]interface{}{"foo": "bar"})

	if wrapped.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, wrapped.Cause)
	}
	if wrapped.I18nKey != "custom.key" {
		t.Errorf("expected key custom.key, got %s", wrapped.I18nKey)
	}
	if wrapped.Technical != "standard error" {
		t.Errorf("expected tech 'standard error', got %s", wrapped.Technical)
	}

	// Test wrapping an already AppError
	appErr := FileNotFoundError("path", nil)
	wrappedAgain := WrapError(appErr, "ignored.key", nil)

	if wrappedAgain != appErr {
		t.Error("WrapError should return the original AppError if it's already one")
	}
}
