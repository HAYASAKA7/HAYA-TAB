//go:build !ios && !android

package main

import (
	"errors"
	"testing"
)

type fakeFileServer struct {
	started  bool
	port     int
	startErr error
}

func (f *fakeFileServer) StartFileServer() (int, error) {
	f.started = true
	return f.port, f.startErr
}

func (f *fakeFileServer) SetFileServerPort(port int) {
	f.port = port
}

func TestConfigureContentTransportStartsDesktopServer(t *testing.T) {
	server := &fakeFileServer{port: 42123}
	if err := configureContentTransport(server); err != nil {
		t.Fatal(err)
	}
	if !server.started || server.port != 42123 {
		t.Fatalf("started=%v port=%d", server.started, server.port)
	}
}

func TestConfigureContentTransportReturnsDesktopServerError(t *testing.T) {
	want := errors.New("listen failed")
	server := &fakeFileServer{startErr: want}

	if err := configureContentTransport(server); !errors.Is(err, want) {
		t.Fatalf("configureContentTransport() error = %v, want %v", err, want)
	}
	if server.port != 0 {
		t.Fatalf("server port = %d, want 0 after startup failure", server.port)
	}
}
