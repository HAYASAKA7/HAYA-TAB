//go:build ios || android

package platform

import "fmt"

func openFileCommand(_ string) (string, []string, error) {
	return "", nil, fmt.Errorf("opening files outside HAYA-TAB is unsupported on %s", CurrentTarget())
}
