package errors

import (
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
	ErrCodeFileNotFound    = "errors.fileNotFound"
	ErrCodeFileAccess      = "errors.fileAccess"
	ErrCodeDuplicate       = "errors.duplicate"
	ErrCodeDatabase        = "errors.database"
	ErrCodeNetwork         = "errors.network"
	ErrCodeWebDAV          = "errors.webdav"
	ErrCodeMetadata        = "errors.metadata"
	ErrCodeCoverDownload   = "errors.coverDownload"
	ErrCodeMusicBrainz     = "errors.musicbrainz"
	ErrCodeInvalidInput    = "errors.invalidInput"
	ErrCodePermission      = "errors.permission"
	ErrCodeUnknown         = "errors.unknown"
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
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return NewAppError(
		i18nKey,
		err.Error(),
		i18nArgs,
		err,
	)
}
