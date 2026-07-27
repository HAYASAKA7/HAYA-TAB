//go:build !ios && !android

package main

type fileServer interface {
	StartFileServer() (int, error)
	SetFileServerPort(int)
}

func configureContentTransport(server fileServer) error {
	port, err := server.StartFileServer()
	if err != nil {
		return err
	}
	server.SetFileServerPort(port)
	return nil
}
