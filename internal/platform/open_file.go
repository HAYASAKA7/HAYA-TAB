package platform

import "os/exec"

// OpenFile asks the native desktop shell to open path with its default app.
// Mobile targets deliberately reject this desktop-only operation.
func OpenFile(path string) error {
	name, args, err := openFileCommand(path)
	if err != nil {
		return err
	}
	return exec.Command(name, args...).Start()
}
