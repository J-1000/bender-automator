package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/user/bender/internal/logging"
)

const DefaultSocketPath = "/tmp/bender.sock"

// JSON-RPC 2.0 structures
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id"`
}

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
	ID      any    `json:"id"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ErrCodeParse          = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

type Handler func(ctx context.Context, params json.RawMessage) (any, error)

type Server struct {
	socketPath string
	listener   net.Listener
	handlers   map[string]Handler
	mu         sync.RWMutex
	wg         sync.WaitGroup
}

func NewServer(socketPath string) *Server {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	return &Server{
		socketPath: socketPath,
		handlers:   make(map[string]Handler),
	}
}

func (s *Server) Handle(method string, handler Handler) {
	s.mu.Lock()
	s.handlers[method] = handler
	s.mu.Unlock()
}

func (s *Server) Start(ctx context.Context) error {
	// Remove existing socket file
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	s.listener = listener

	// Set socket permissions
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		listener.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}

	logging.Info("API server listening on %s", s.socketPath)

	go s.acceptLoop(ctx)

	return nil
}

func (s *Server) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				logging.Error("accept connection: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConnection(ctx, conn)
		}()
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	encoder := json.NewEncoder(conn)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := Response{
				JSONRPC: "2.0",
				Error: &Error{
					Code:    ErrCodeParse,
					Message: "Parse error",
					Data:    err.Error(),
				},
				ID: nil,
			}
			encoder.Encode(resp)
			continue
		}

		resp := s.handleRequest(ctx, &req)
		if err := encoder.Encode(resp); err != nil {
			logging.Error("encode response: %v", err)
			return
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req *Request) Response {
	if req.JSONRPC != "2.0" {
		return Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    ErrCodeInvalidRequest,
				Message: "Invalid Request",
				Data:    "jsonrpc must be 2.0",
			},
			ID: req.ID,
		}
	}

	s.mu.RLock()
	handler, ok := s.handlers[req.Method]
	s.mu.RUnlock()

	if !ok {
		return Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    ErrCodeMethodNotFound,
				Message: "Method not found",
				Data:    req.Method,
			},
			ID: req.ID,
		}
	}

	result, err := handler(ctx, req.Params)
	if err != nil {
		return Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    ErrCodeInternal,
				Message: err.Error(),
			},
			ID: req.ID,
		}
	}

	return Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

func (s *Server) Stop() error {
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	os.Remove(s.socketPath)
	logging.Info("API server stopped")
	return nil
}

func (s *Server) SocketPath() string {
	return s.socketPath
}
