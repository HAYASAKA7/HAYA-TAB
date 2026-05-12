package errors

import (
	stderrors "errors"
	"fmt"
)

// AppError represents an application error with both technical and i18n key
type AppError struct {
	Technical string                 // Technical error message for logging
	I18nKey   string                 // i18n key for frontend translation
	I18nArgs  map[string]interface{} // Arguments for i18n interpolation
	Code      string                 // Error code for categorization
	Cause     error                  // Original error for debugging
}

// Error implements the error interface (returns technical message for logging)
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Technical, e.Cause)
	}
	return e.Technical
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// I18nData returns the i18n key and arguments for frontend translation
func (e *AppError) I18nData() (string, map[string]interface{}) {
	return e.I18nKey, e.I18nArgs
}

// Error codes for categorization (used as i18n keys)
const (
	ErrCodeFileNotFound  = "errors.fileNotFound"
	ErrCodeFileAccess    = "errors.fileAccess"
	ErrCodeDuplicate     = "errors.duplicate"
	ErrCodeDatabase      = "errors.database"
	ErrCodeNetwork       = "errors.network"
	ErrCodeWebDAV        = "errors.webdav"
	ErrCodeMetadata      = "errors.metadata"
	ErrCodeCoverDownload = "errors.coverDownload"
	ErrCodeMusicBrainz   = "errors.musicbrainz"
	ErrCodeInvalidInput  = "errors.invalidInput"
	ErrCodePermission    = "errors.permission"
	ErrCodeUnknown       = "errors.unknown"
)

// NewAppError creates a new application error
func NewAppError(i18nKey, technical string, i18nArgs map[string]interface{}, cause error) *AppError {
	return &AppError{
		Code:      i18nKey,
		I18nKey:   i18nKey,
		Technical: technical,
		I18nArgs:  i18nArgs,
		Cause:     cause,
	}
}

// Common error constructors for better consistency

// FileNotFoundError creates an error for missing files
func FileNotFoundError(path string, cause error) *AppError {
	return NewAppError(
		ErrCodeFileNotFound,
		fmt.Sprintf("file not found: %s", path),
		map[string]interface{}{"path": path},
		cause,
	)
}

// FileAccessError creates an error for file access issues
func FileAccessError(path string, cause error) *AppError {
	return NewAppError(
		ErrCodeFileAccess,
		fmt.Sprintf("failed to access file: %s", path),
		map[string]interface{}{"path": path},
		cause,
	)
}

// DuplicateError creates an error for duplicate entries
func DuplicateError(itemType, name string) *AppError {
	return NewAppError(
		ErrCodeDuplicate,
		fmt.Sprintf("duplicate %s: %s", itemType, name),
		map[string]interface{}{"type": itemType, "name": name},
		nil,
	)
}

// DatabaseError creates an error for database operations
func DatabaseError(operation string, cause error) *AppError {
	return NewAppError(
		ErrCodeDatabase,
		fmt.Sprintf("database %s failed", operation),
		map[string]interface{}{"operation": operation},
		cause,
	)
}

// NetworkError creates an error for network issues
func NetworkError(operation string, cause error) *AppError {
	return NewAppError(
		ErrCodeNetwork,
		fmt.Sprintf("network error during %s", operation),
		map[string]interface{}{"operation": operation},
		cause,
	)
}

// WebDAVError creates an error for WebDAV operations
func WebDAVError(operation string, cause error) *AppError {
	return NewAppError(
		ErrCodeWebDAV,
		fmt.Sprintf("WebDAV %s failed", operation),
		map[string]interface{}{"operation": operation},
		cause,
	)
}

// MetadataError creates an error for metadata parsing
func MetadataError(filename string, cause error) *AppError {
	return NewAppError(
		ErrCodeMetadata,
		fmt.Sprintf("failed to parse metadata from %s", filename),
		map[string]interface{}{"filename": filename},
		cause,
	)
}

// CoverDownloadError creates an error for cover art download failures
func CoverDownloadError(artist, album string, cause error) *AppError {
	return NewAppError(
		ErrCodeCoverDownload,
		fmt.Sprintf("failed to download cover for %s - %s", artist, album),
		map[string]interface{}{"artist": artist, "album": album},
		cause,
	)
}

// MusicBrainzError creates an error for MusicBrainz API failures
func MusicBrainzError(artist string, cause error) *AppError {
	return NewAppError(
		ErrCodeMusicBrainz,
		fmt.Sprintf("MusicBrainz lookup failed for %s", artist),
		map[string]interface{}{"artist": artist},
		cause,
	)
}

// InvalidInputError creates an error for invalid user input
func InvalidInputError(field, reason string) *AppError {
	return NewAppError(
		ErrCodeInvalidInput,
		fmt.Sprintf("invalid input for %s: %s", field, reason),
		map[string]interface{}{"field": field, "reason": reason},
		nil,
	)
}

// PermissionError creates an error for permission issues
func PermissionError(operation string, cause error) *AppError {
	return NewAppError(
		ErrCodePermission,
		fmt.Sprintf("permission denied for %s", operation),
		map[string]interface{}{"operation": operation},
		cause,
	)
}

// WrapError wraps an existing error with i18n key
func WrapError(err error, i18nKey string, i18nArgs map[string]interface{}) *AppError {
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr
	}
	return NewAppError(
		i18nKey,
		err.Error(),
		i18nArgs,
		err,
	)
}

// ErrorPayload represents a serializable error for API responses
type ErrorPayload struct {
	Code      string                 `json:"code"`
	I18nKey   string                 `json:"i18nKey"`
	I18nArgs  map[string]interface{} `json:"i18nArgs,omitempty"`
	Message   string                 `json:"message"`
	Technical string                 `json:"technical,omitempty"`
}

// ToPayload converts any error to ErrorPayload
func ToPayload(err error) ErrorPayload {
	if err == nil {
		return ErrorPayload{
			Code:    ErrCodeUnknown,
			I18nKey: ErrCodeUnknown,
			Message: "unknown error",
		}
	}

	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return ErrorPayload{
			Code:      appErr.Code,
			I18nKey:   appErr.I18nKey,
			I18nArgs:  appErr.I18nArgs,
			Message:   appErr.Technical,
			Technical: appErr.Technical,
		}
	}

	return ErrorPayload{
		Code:      ErrCodeUnknown,
		I18nKey:   ErrCodeUnknown,
		Message:   err.Error(),
		Technical: err.Error(),
	}
}

// TabNotFoundError creates an error for missing tabs
func TabNotFoundError(id string) *AppError {
	return NewAppError(
		"errors.tabNotFound",
		fmt.Sprintf("tab not found: %s", id),
		map[string]interface{}{"id": id},
		nil,
	)
}

// TabDuplicateError creates an error for duplicate tabs
func TabDuplicateError(name string) *AppError {
	return NewAppError(
		"errors.tabDuplicate",
		fmt.Sprintf("tab already exists: %s", name),
		map[string]interface{}{"name": name},
		nil,
	)
}

// CategoryMoveToSelfError creates an error when trying to move a category into itself
func CategoryMoveToSelfError() *AppError {
	return NewAppError(
		"errors.categoryMoveToSelf",
		"cannot move category into itself",
		nil,
		nil,
	)
}

// WebDAVNotEnabledError creates an error when WebDAV operations are attempted without WebDAV enabled
func WebDAVNotEnabledError() *AppError {
	return NewAppError(
		"errors.webdavNotEnabled",
		"WebDAV is not enabled",
		nil,
		nil,
	)
}

// WebDAVDiscoverVolumesError creates an error when volume discovery fails
func WebDAVDiscoverVolumesError(cause error) *AppError {
	return NewAppError(
		"errors.webdavDiscoverVolumesFailed",
		"failed to discover WebDAV volumes",
		nil,
		cause,
	)
}

// WebDAVConnectionTestFailedError creates an error when WebDAV connection test fails
func WebDAVConnectionTestFailedError() *AppError {
	return NewAppError(
		"errors.webdavConnectionTestFailed",
		"WebDAV connection test failed",
		nil,
		nil,
	)
}

// WebDAVVolumeHealthCheckError creates an error when volume health check fails
func WebDAVVolumeHealthCheckError(cause error) *AppError {
	return NewAppError(
		"errors.webdavVolumeHealthCheckFailed",
		"failed to check volume health",
		nil,
		cause,
	)
}

// TabNotCloudError creates an error when a cloud operation is attempted on a non-cloud tab
func TabNotCloudError() *AppError {
	return NewAppError(
		"errors.tabNotCloud",
		"tab is not a cloud tab",
		nil,
		nil,
	)
}

// MigrationTargetError creates an error for invalid migration target
func MigrationTargetError(target string) *AppError {
	return NewAppError(
		"errors.migrationInvalidTarget",
		fmt.Sprintf("invalid migration target: %s", target),
		map[string]interface{}{"target": target},
		nil,
	)
}

// MigrationDiskSpaceError creates an error when there's not enough disk space
func MigrationDiskSpaceError(required, available uint64) *AppError {
	return NewAppError(
		"errors.migrationInsufficientDiskSpace",
		fmt.Sprintf("insufficient disk space: required %d bytes, available %d bytes", required, available),
		map[string]interface{}{"required": required, "available": available},
		nil,
	)
}

// MigrationFileCopyError creates an error when file copy fails during migration
func MigrationFileCopyError(filename string, cause error) *AppError {
	return NewAppError(
		"errors.migrationFileCopyFailed",
		fmt.Sprintf("failed to copy file %s", filename),
		map[string]interface{}{"filename": filename},
		cause,
	)
}

// MigrationSizeMismatchError creates an error when migration verification fails
func MigrationSizeMismatchError(expected, actual int64) *AppError {
	return NewAppError(
		"errors.migrationSizeMismatch",
		fmt.Sprintf("migration verification failed: expected %d bytes, got %d bytes", expected, actual),
		map[string]interface{}{"expected": expected, "actual": actual},
		nil,
	)
}

// PluginManagerNotInitializedError creates an error when plugin operations are attempted before initialization
func PluginManagerNotInitializedError() *AppError {
	return NewAppError(
		"errors.pluginManagerNotInitialized",
		"plugin manager not initialized",
		nil,
		nil,
	)
}

// PluginNotFoundError creates an error when a plugin cannot be found
func PluginNotFoundError(id string) *AppError {
	return NewAppError(
		"errors.pluginNotFound",
		fmt.Sprintf("plugin not found: %s", id),
		map[string]interface{}{"id": id},
		nil,
	)
}

// InvalidVolumeNameError creates an error for empty or invalid volume names
func InvalidVolumeNameError() *AppError {
	return NewAppError(
		"errors.invalidVolumeName",
		"volume name cannot be empty",
		nil,
		nil,
	)
}

// VolumeCreationError creates an error when WebDAV volume creation fails
func VolumeCreationError(cause error) *AppError {
	return NewAppError(
		"errors.volumeCreateFailed",
		"failed to create WebDAV volume",
		nil,
		cause,
	)
}

// VolumeRegistrationError creates an error when cloud volume registration fails
func VolumeRegistrationError(cause error) *AppError {
	return NewAppError(
		"errors.volumeRegisterFailed",
		"failed to register cloud volume",
		nil,
		cause,
	)
}
