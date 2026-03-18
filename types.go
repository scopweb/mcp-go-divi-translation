package main

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// MCP JSON-RPC types
const MCP_PROTOCOL_VERSION = "2025-11-25"

// Standard JSON-RPC 2.0 error codes
const (
	ErrParseError     = -32700 // Invalid JSON received
	ErrInvalidRequest = -32600 // Invalid JSON-RPC request
	ErrMethodNotFound = -32601 // Method does not exist
	ErrInvalidParams  = -32602 // Invalid method parameters
	ErrInternalError  = -32603 // Internal server error
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    struct {
		Tools map[string]interface{} `json:"tools"`
	} `json:"capabilities"`
	ServerInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// TranslationSession holds the state for an ongoing translation (legacy chunk-by-chunk)
type TranslationSession struct {
	SourceType   string // "file" or "wordpress"
	InputPath    string
	OutputPath   string
	PostID       int64
	BackupPath   string
	TargetLang   string
	Tokens       []Token
	TextChunks   []string
	ChunkIndices []int
	Translations []string
	CurrentChunk int
	TotalChunks  int
}

// BulkTranslationSession holds state for the optimized bulk translation flow
type BulkTranslationSession struct {
	ExtractionID string
	SourceType   string
	InputPath    string
	OutputPath   string
	PostID       int64
	BackupPath   string
	TargetLang   string
	Tokens       []Token
	ChunkIndices []int
	TotalChunks  int
	Parts        int
	CurrentPart  int
	PartRanges   [][2]int
	Translations []string
	TextForTranslation string
	CreatedAt    time.Time
	// WordPress metadata
	OriginalTitle     string
	OriginalSlug      string
	OriginalExcerpt   string
	TranslatedTitle   string
	TranslatedSlug    string
	TranslatedExcerpt string
}

// Global storage for active extraction sessions
var (
	activeExtractions = make(map[string]*BulkTranslationSession)
	extractionsMutex  sync.RWMutex
)


// MCPServer implements the MCP protocol
type MCPServer struct {
	stdin          io.Reader
	stdout         io.Writer
	stderr         io.Writer
	session        *TranslationSession
	bulkSession    *BulkTranslationSession
	wpDB           *WordPressDB
	shouldShutdown bool
	initialized    bool
}
