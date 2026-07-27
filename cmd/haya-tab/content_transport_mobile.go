//go:build ios || android

package main

type fileServer interface {
	StartFileServer() (int, error)
	SetFileServerPort(int)
}

func configureContentTransport(_ fileServer) error {
	return nil
}
