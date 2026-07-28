//go:build !ios && !android

package platform

import (
	"fmt"
	"runtime"
)

func openFileCommand(path string) (string, []string, error) {
	switch runtime.GOOS {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", path}, nil
	case "darwin":
		return "open", []string{path}, nil
	case "linux":
		return "xdg-open", []string{path}, nil
	default:
		return "", nil, fmt.Errorf("opening files is unsupported on %s", runtime.GOOS)
	}
}
