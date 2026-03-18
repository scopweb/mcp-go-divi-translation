package main

import (
	"bufio"
	"encoding/json"
	"fmt"
)

// Run starts the MCP server main loop, reading JSON-RPC requests from stdin
func (s *MCPServer) Run() {
	scanner := bufio.NewScanner(s.stdin)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.log("Error parsing request: %v", err)
			s.writeResponse(s.rpcError(ErrParseError, fmt.Sprintf("Parse error: %v", err), nil))
			continue
		}

		// Validate request ID per MCP spec
		if req.ID != nil && !isValidRequestID(req.ID) {
			s.writeResponse(s.rpcError(ErrInvalidRequest, "Invalid Request: ID must be a string or integer, not null or other types", nil))
			continue
		}

		s.log("Received method: %s", req.Method)

		// Initialization gate: only allow initialize before handshake completes
		if !s.initialized && req.Method != "initialize" {
			if req.ID != nil {
				s.writeResponse(s.rpcError(ErrInvalidRequest, "Server not initialized. Send 'initialize' first.", req.ID))
			}
			continue
		}

		switch req.Method {
		case "initialize":
			s.handleInitialize(req)
		case "shutdown":
			s.handleShutdown(req)
			return
		case "tools/list":
			s.handleListTools(req)
		case "tools/call":
			s.handleCallTool(req)
		case "ping":
			s.handlePing(req)
		case "notifications/initialized":
			// Client notification, no response needed
		default:
			if req.ID != nil {
				s.writeResponse(s.rpcError(ErrMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), req.ID))
			}
		}

		if s.shouldShutdown {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		s.log("Scanner error: %v", err)
	}

	// Clean up WordPress connection
	if s.wpDB != nil {
		s.wpDB.Close()
	}
}
