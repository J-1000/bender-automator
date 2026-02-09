package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

func testSocket(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("/tmp/bender-test-%d.sock", os.Getpid())
}

func rpcCall(t *testing.T, sock, method string, params any) Response {
	t.Helper()
	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	var rawParams json.RawMessage
	if params != nil {
		rawParams, _ = json.Marshal(params)
	}

	req := Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
		ID:      1,
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

func TestServerStartStop(t *testing.T) {
	sock := testSocket(t)
	defer os.Remove(sock)

	s := NewServer(sock)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Verify socket file exists
	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("socket file not found: %v", err)
	}

	s.Stop()

	// Verify socket cleaned up
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Fatal("socket file should be removed after stop")
	}
}

func TestServerHandleMethod(t *testing.T) {
	sock := testSocket(t)
	defer os.Remove(sock)

	s := NewServer(sock)
	s.Handle("test.ping", func(ctx context.Context, params json.RawMessage) (any, error) {
		return map[string]string{"pong": "ok"}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	time.Sleep(10 * time.Millisecond) // let server start accepting

	resp := rpcCall(t, sock, "test.ping", nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	result, _ := json.Marshal(resp.Result)
	var m map[string]string
	json.Unmarshal(result, &m)
	if m["pong"] != "ok" {
		t.Fatalf("expected pong=ok, got %v", m)
	}
}

func TestServerMethodNotFound(t *testing.T) {
	sock := testSocket(t)
	defer os.Remove(sock)

	s := NewServer(sock)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	time.Sleep(10 * time.Millisecond)

	resp := rpcCall(t, sock, "nonexistent.method", nil)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Fatalf("expected code %d, got %d", ErrCodeMethodNotFound, resp.Error.Code)
	}
}

func TestServerHandlerError(t *testing.T) {
	sock := testSocket(t)
	defer os.Remove(sock)

	s := NewServer(sock)
	s.Handle("test.fail", func(ctx context.Context, params json.RawMessage) (any, error) {
		return nil, fmt.Errorf("something went wrong")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	time.Sleep(10 * time.Millisecond)

	resp := rpcCall(t, sock, "test.fail", nil)
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Message != "something went wrong" {
		t.Fatalf("expected error message, got %q", resp.Error.Message)
	}
}

func TestServerParamsPassthrough(t *testing.T) {
	sock := testSocket(t)
	defer os.Remove(sock)

	s := NewServer(sock)
	s.Handle("test.echo", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return map[string]string{"echo": p.Message}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	time.Sleep(10 * time.Millisecond)

	resp := rpcCall(t, sock, "test.echo", map[string]string{"message": "hello"})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	result, _ := json.Marshal(resp.Result)
	var m map[string]string
	json.Unmarshal(result, &m)
	if m["echo"] != "hello" {
		t.Fatalf("expected echo=hello, got %v", m)
	}
}

func TestStatusHandlers(t *testing.T) {
	sock := testSocket(t)
	defer os.Remove(sock)

	s := NewServer(sock)
	RegisterStatusHandlers(s, "1.0.0-test")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	time.Sleep(10 * time.Millisecond)

	// Test status.get
	resp := rpcCall(t, sock, "status.get", nil)
	if resp.Error != nil {
		t.Fatalf("status.get error: %s", resp.Error.Message)
	}

	result, _ := json.Marshal(resp.Result)
	var status DaemonStatus
	json.Unmarshal(result, &status)
	if !status.Running {
		t.Fatal("expected running=true")
	}
	if status.Version != "1.0.0-test" {
		t.Fatalf("expected version=1.0.0-test, got %s", status.Version)
	}
	if status.PID == 0 {
		t.Fatal("expected non-zero PID")
	}

	// Test status.health
	resp = rpcCall(t, sock, "status.health", nil)
	if resp.Error != nil {
		t.Fatalf("status.health error: %s", resp.Error.Message)
	}

	result, _ = json.Marshal(resp.Result)
	var health HealthCheck
	json.Unmarshal(result, &health)
	if health.Status != "healthy" {
		t.Fatalf("expected status=healthy, got %s", health.Status)
	}
	if health.Checks["daemon"] != "ok" {
		t.Fatalf("expected daemon check=ok, got %s", health.Checks["daemon"])
	}
}

func TestMultipleRequests(t *testing.T) {
	sock := testSocket(t)
	defer os.Remove(sock)

	callCount := 0
	s := NewServer(sock)
	s.Handle("test.count", func(ctx context.Context, params json.RawMessage) (any, error) {
		callCount++
		return map[string]int{"count": callCount}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	time.Sleep(10 * time.Millisecond)

	// Make multiple calls from separate connections
	for i := 1; i <= 3; i++ {
		resp := rpcCall(t, sock, "test.count", nil)
		if resp.Error != nil {
			t.Fatalf("call %d error: %s", i, resp.Error.Message)
		}
	}

	if callCount != 3 {
		t.Fatalf("expected 3 calls, got %d", callCount)
	}
}
