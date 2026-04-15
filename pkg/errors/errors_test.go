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

	appErrNoCause := &AppError{Technical: "technical message"}
	if appErrNoCause.Error() != "technical message" {
		t.Errorf("expected 'technical message', got %q", appErrNoCause.Error())
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	appErr := &AppError{Cause: cause}
	if appErr.Unwrap() != cause {
		t.Errorf("expected %v, got %v", cause, appErr.Unwrap())
	}
}

func TestAppError_I18nData(t *testing.T) {
	args := map[string]interface{}{"key": "value"}
	appErr := &AppError{I18nKey: "test.key", I18nArgs: args}

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

// TestErrorConstructors covers the generic constructors.
func TestErrorConstructors(t *testing.T) {
	cause := errors.New("io error")

	tests := []struct {
		name     string
		err      *AppError
		wantKey  string
		wantTech string
	}{
		{"FileNotFound", FileNotFoundError("path/to/file", cause), ErrCodeFileNotFound, "file not found: path/to/file"},
		{"FileAccess", FileAccessError("path/to/file", cause), ErrCodeFileAccess, "failed to access file: path/to/file"},
		{"Duplicate", DuplicateError("category", "rock"), ErrCodeDuplicate, "duplicate category: rock"},
		{"Database", DatabaseError("insert", cause), ErrCodeDatabase, "database insert failed"},
		{"Network", NetworkError("GET", cause), ErrCodeNetwork, "network error during GET"},
		{"WebDAV", WebDAVError("PROPFIND", cause), ErrCodeWebDAV, "WebDAV PROPFIND failed"},
		{"Metadata", MetadataError("file.mp3", cause), ErrCodeMetadata, "failed to parse metadata from file.mp3"},
		{"CoverDownload", CoverDownloadError("Artist", "Album", cause), ErrCodeCoverDownload, "failed to download cover for Artist - Album"},
		{"MusicBrainz", MusicBrainzError("Artist", cause), ErrCodeMusicBrainz, "MusicBrainz lookup failed for Artist"},
		{"InvalidInput", InvalidInputError("field1", "too long"), ErrCodeInvalidInput, "invalid input for field1: too long"},
		{"Permission", PermissionError("delete", cause), ErrCodePermission, "permission denied for delete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.I18nKey != tt.wantKey {
				t.Errorf("I18nKey = %q, want %q", tt.err.I18nKey, tt.wantKey)
			}
			if tt.err.Technical != tt.wantTech {
				t.Errorf("Technical = %q, want %q", tt.err.Technical, tt.wantTech)
			}
		})
	}
}

// TestDomainErrorConstructors covers the app-domain-specific constructors
// that were not exercised by TestErrorConstructors.
func TestDomainErrorConstructors(t *testing.T) {
	cause := errors.New("underlying error")

	t.Run("TabNotFoundError", func(t *testing.T) {
		err := TabNotFoundError("tab-123")
		if err.I18nKey != "errors.tabNotFound" {
			t.Errorf("I18nKey = %q, want errors.tabNotFound", err.I18nKey)
		}
		if err.Technical != "tab not found: tab-123" {
			t.Errorf("Technical = %q", err.Technical)
		}
		if err.I18nArgs["id"] != "tab-123" {
			t.Errorf("I18nArgs[id] = %v", err.I18nArgs["id"])
		}
		if err.Cause != nil {
			t.Error("expected nil cause")
		}
	})

	t.Run("TabDuplicateError", func(t *testing.T) {
		err := TabDuplicateError("Smoke on the Water")
		if err.I18nKey != "errors.tabDuplicate" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.I18nArgs["name"] != "Smoke on the Water" {
			t.Errorf("I18nArgs[name] = %v", err.I18nArgs["name"])
		}
	})

	t.Run("CategoryMoveToSelfError", func(t *testing.T) {
		err := CategoryMoveToSelfError()
		if err.I18nKey != "errors.categoryMoveToSelf" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.Technical != "cannot move category into itself" {
			t.Errorf("Technical = %q", err.Technical)
		}
	})

	t.Run("WebDAVNotEnabledError", func(t *testing.T) {
		err := WebDAVNotEnabledError()
		if err.I18nKey != "errors.webdavNotEnabled" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.Technical != "WebDAV is not enabled" {
			t.Errorf("Technical = %q", err.Technical)
		}
	})

	t.Run("WebDAVDiscoverVolumesError", func(t *testing.T) {
		err := WebDAVDiscoverVolumesError(cause)
		if err.I18nKey != "errors.webdavDiscoverVolumesFailed" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.Cause != cause {
			t.Error("Cause should be set")
		}
	})

	t.Run("WebDAVConnectionTestFailedError", func(t *testing.T) {
		err := WebDAVConnectionTestFailedError()
		if err.I18nKey != "errors.webdavConnectionTestFailed" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
	})

	t.Run("WebDAVVolumeHealthCheckError", func(t *testing.T) {
		err := WebDAVVolumeHealthCheckError(cause)
		if err.I18nKey != "errors.webdavVolumeHealthCheckFailed" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.Cause != cause {
			t.Error("Cause should be set")
		}
	})

	t.Run("TabNotCloudError", func(t *testing.T) {
		err := TabNotCloudError()
		if err.I18nKey != "errors.tabNotCloud" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
	})

	t.Run("MigrationTargetError", func(t *testing.T) {
		err := MigrationTargetError("/some/path")
		if err.I18nKey != "errors.migrationInvalidTarget" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.I18nArgs["target"] != "/some/path" {
			t.Errorf("I18nArgs[target] = %v", err.I18nArgs["target"])
		}
	})

	t.Run("MigrationDiskSpaceError", func(t *testing.T) {
		err := MigrationDiskSpaceError(uint64(1024), uint64(512))
		if err.I18nKey != "errors.migrationInsufficientDiskSpace" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.I18nArgs["required"] != uint64(1024) {
			t.Errorf("I18nArgs[required] = %v", err.I18nArgs["required"])
		}
		if err.I18nArgs["available"] != uint64(512) {
			t.Errorf("I18nArgs[available] = %v", err.I18nArgs["available"])
		}
	})

	t.Run("MigrationFileCopyError", func(t *testing.T) {
		err := MigrationFileCopyError("song.pdf", cause)
		if err.I18nKey != "errors.migrationFileCopyFailed" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.I18nArgs["filename"] != "song.pdf" {
			t.Errorf("I18nArgs[filename] = %v", err.I18nArgs["filename"])
		}
		if err.Cause != cause {
			t.Error("Cause should be set")
		}
	})

	t.Run("MigrationSizeMismatchError", func(t *testing.T) {
		err := MigrationSizeMismatchError(int64(100), int64(90))
		if err.I18nKey != "errors.migrationSizeMismatch" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.I18nArgs["expected"] != int64(100) {
			t.Errorf("I18nArgs[expected] = %v", err.I18nArgs["expected"])
		}
		if err.I18nArgs["actual"] != int64(90) {
			t.Errorf("I18nArgs[actual] = %v", err.I18nArgs["actual"])
		}
	})

	t.Run("PluginManagerNotInitializedError", func(t *testing.T) {
		err := PluginManagerNotInitializedError()
		if err.I18nKey != "errors.pluginManagerNotInitialized" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.Technical != "plugin manager not initialized" {
			t.Errorf("Technical = %q", err.Technical)
		}
	})

	t.Run("PluginNotFoundError", func(t *testing.T) {
		err := PluginNotFoundError("my-plugin")
		if err.I18nKey != "errors.pluginNotFound" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.I18nArgs["id"] != "my-plugin" {
			t.Errorf("I18nArgs[id] = %v", err.I18nArgs["id"])
		}
	})

	t.Run("InvalidVolumeNameError", func(t *testing.T) {
		err := InvalidVolumeNameError()
		if err.I18nKey != "errors.invalidVolumeName" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.Technical != "volume name cannot be empty" {
			t.Errorf("Technical = %q", err.Technical)
		}
	})

	t.Run("VolumeCreationError", func(t *testing.T) {
		err := VolumeCreationError(cause)
		if err.I18nKey != "errors.volumeCreateFailed" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.Cause != cause {
			t.Error("Cause should be set")
		}
	})

	t.Run("VolumeRegistrationError", func(t *testing.T) {
		err := VolumeRegistrationError(cause)
		if err.I18nKey != "errors.volumeRegisterFailed" {
			t.Errorf("I18nKey = %q", err.I18nKey)
		}
		if err.Cause != cause {
			t.Error("Cause should be set")
		}
	})
}

func TestWrapError(t *testing.T) {
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

	// An already-AppError should pass through unchanged.
	appErr := FileNotFoundError("path", nil)
	wrappedAgain := WrapError(appErr, "ignored.key", nil)
	if wrappedAgain != appErr {
		t.Error("WrapError should return the original AppError unchanged")
	}
}

// TestToPayload exercises all three branches of the ToPayload function.
func TestToPayload(t *testing.T) {
	t.Run("nil error returns unknown placeholder", func(t *testing.T) {
		p := ToPayload(nil)
		if p.Code != ErrCodeUnknown {
			t.Errorf("Code = %q, want %q", p.Code, ErrCodeUnknown)
		}
		if p.I18nKey != ErrCodeUnknown {
			t.Errorf("I18nKey = %q, want %q", p.I18nKey, ErrCodeUnknown)
		}
		if p.Message != "unknown error" {
			t.Errorf("Message = %q, want 'unknown error'", p.Message)
		}
	})

	t.Run("AppError is faithfully converted", func(t *testing.T) {
		appErr := FileNotFoundError("test.pdf", nil)
		p := ToPayload(appErr)
		if p.Code != ErrCodeFileNotFound {
			t.Errorf("Code = %q, want %q", p.Code, ErrCodeFileNotFound)
		}
		if p.I18nKey != ErrCodeFileNotFound {
			t.Errorf("I18nKey = %q, want %q", p.I18nKey, ErrCodeFileNotFound)
		}
		if p.I18nArgs["path"] != "test.pdf" {
			t.Errorf("I18nArgs[path] = %v, want test.pdf", p.I18nArgs["path"])
		}
		if p.Technical == "" {
			t.Error("Technical should not be empty")
		}
		if p.Message == "" {
			t.Error("Message should not be empty")
		}
	})

	t.Run("plain error returns unknown code with message", func(t *testing.T) {
		plain := errors.New("something went wrong")
		p := ToPayload(plain)
		if p.Code != ErrCodeUnknown {
			t.Errorf("Code = %q, want %q", p.Code, ErrCodeUnknown)
		}
		if p.Message != "something went wrong" {
			t.Errorf("Message = %q, want 'something went wrong'", p.Message)
		}
		if p.Technical != "something went wrong" {
			t.Errorf("Technical = %q, want 'something went wrong'", p.Technical)
		}
		// plain errors should have no I18nArgs
		if len(p.I18nArgs) != 0 {
			t.Errorf("I18nArgs should be nil/empty for plain errors, got %v", p.I18nArgs)
		}
	})
}
