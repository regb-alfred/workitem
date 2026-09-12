package driverprotocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

type BootstrapServer struct {
	listener net.Listener
	path     string
}

func Listen(path string) (*BootstrapServer, error) {
	if path == "" {
		return nil, fmt.Errorf("driver bootstrap socket path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		listener.Close()
		_ = os.Remove(path)
		return nil, err
	}
	return &BootstrapServer{listener: listener, path: path}, nil
}

func (s *BootstrapServer) Close() error {
	err := s.listener.Close()
	_ = os.Remove(s.path)
	return err
}

func (s *BootstrapServer) ServeRequestOnce(ctx context.Context, request Request) (Response, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if listener, ok := s.listener.(*net.UnixListener); ok {
			_ = listener.SetDeadline(deadline)
		}
	}
	conn, err := s.listener.Accept()
	if err != nil {
		if ctx.Err() != nil {
			return Response{}, ctx.Err()
		}
		return Response{}, err
	}
	defer conn.Close()
	var hello Hello
	if err := json.NewDecoder(conn).Decode(&hello); err != nil {
		return Response{}, fmt.Errorf("read driver hello: %w", err)
	}
	if hello.ProtocolVersion != Version {
		return Response{}, fmt.Errorf("driver protocol %d is incompatible with %d", hello.ProtocolVersion, Version)
	}
	if hello.Driver != request.Driver {
		return Response{}, fmt.Errorf("driver identified as %q, expected %q", hello.Driver, request.Driver)
	}
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return Response{}, fmt.Errorf("send driver request: %w", err)
	}
	var response Response
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return Response{}, fmt.Errorf("read driver response: %w", err)
	}
	if response.ProtocolVersion != Version {
		return Response{}, fmt.Errorf("driver response protocol %d is incompatible with %d", response.ProtocolVersion, Version)
	}
	if response.Error != "" {
		return response, fmt.Errorf("driver rejected %s: %s", request.Operation, response.Error)
	}
	return response, nil
}

func (s *BootstrapServer) ServeOnce(ctx context.Context, request LaunchRequest) (Ready, error) {
	response, err := s.ServeRequestOnce(ctx, Request{ProtocolVersion: Version, Driver: request.Driver, Operation: OperationLaunch, Launch: &request})
	if err != nil {
		return Ready{}, err
	}
	if response.Ready == nil {
		return Ready{}, fmt.Errorf("driver launch response omitted readiness")
	}
	return *response.Ready, nil
}

func ConnectRequest(ctx context.Context, path string, hello Hello) (Request, func(Response) error, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", path)
	if err != nil {
		return Request{}, nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := json.NewEncoder(conn).Encode(hello); err != nil {
		conn.Close()
		return Request{}, nil, err
	}
	var request Request
	if err := json.NewDecoder(conn).Decode(&request); err != nil {
		conn.Close()
		return Request{}, nil, err
	}
	reply := func(response Response) error {
		defer conn.Close()
		return json.NewEncoder(conn).Encode(response)
	}
	return request, reply, nil
}

func Connect(ctx context.Context, path string, hello Hello) (LaunchRequest, func(Ready) error, error) {
	request, respond, err := ConnectRequest(ctx, path, hello)
	if err != nil {
		return LaunchRequest{}, nil, err
	}
	if request.Operation != OperationLaunch || request.Launch == nil {
		_ = respond(Response{ProtocolVersion: Version, Error: "expected launch operation"})
		return LaunchRequest{}, nil, fmt.Errorf("expected launch operation")
	}
	reply := func(ready Ready) error {
		return respond(Response{ProtocolVersion: Version, Ready: &ready, Error: ready.Error})
	}
	return *request.Launch, reply, nil
}
