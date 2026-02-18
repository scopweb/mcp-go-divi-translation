# Future Improvements Roadmap
## MCP Divi Translator - Beyond 2025-11-25 Compliance

**Status**: Post-compliance recommendations
**Difficulty**: Medium to High
**Target Version**: v4.4.0+

---

## Priority 1: Capabilities Negotiation (Medium)

### Current State
Server declares capabilities but doesn't validate client capabilities.

### Implementation Goal
Proper client-server capability negotiation per MCP spec.

### Changes Required

1. **Store client capabilities during initialize**
   ```go
   type MCPServer struct {
       // ... existing fields
       clientCapabilities map[string]interface{} // NEW
   }
   ```

2. **Validate in handleInitialize()**
   ```go
   var params InitializeParams
   json.Unmarshal(req.Params, &params)
   s.clientCapabilities = params.Capabilities
   ```

3. **Check capabilities before operations**
   ```go
   if s.clientCapabilities["tools"] != true {
       // Handle client that doesn't support tools
   }
   ```

### Files to Modify
- `mcp_server.go` (InitializeParams, handleInitialize, MCPServer struct)

### Testing
- Test with client that has no tools capability
- Test with client that has partial capabilities
- Verify graceful degradation

### Estimated Effort
- Implementation: 2 hours
- Testing: 1 hour
- Documentation: 30 minutes

---

## Priority 2: Standard Error Codes (Medium)

### Current State
Basic error codes (-32602, -32601) used but not comprehensive.

### Implementation Goal
Map all tool errors to standard MCP error codes.

### Error Code Reference
```
-32700: Parse error
-32600: Invalid Request
-32601: Method not found
-32602: Invalid params
-32603: Internal error
-32000 to -32099: Server error (reserved)
```

### Implementation Steps

1. **Create error code constants**
   ```go
   const (
       ErrParseError     = -32700
       ErrInvalidRequest = -32600
       ErrMethodNotFound = -32601
       ErrInvalidParams  = -32602
       ErrInternalError  = -32603
       ErrServerError    = -32000
   )
   ```

2. **Add error helper function**
   ```go
   func (s *MCPServer) errorResponse(code int, message string, id interface{}) JSONRPCResponse {
       return JSONRPCResponse{
           JSONRPC: "2.0",
           ID:      id,
           Error: &RPCError{
               Code:    code,
               Message: message,
           },
       }
   }
   ```

3. **Update all error responses**
   - File read errors → -32603 (Internal error)
   - Invalid arguments → -32602 (Invalid params)
   - Missing files → -32603 (Internal error)
   - WordPress connection → -32603 (Internal error)

### Files to Modify
- `mcp_server.go` (all error handling)

### Testing
- Verify each error code is used correctly
- Test error messages are descriptive
- Verify client can distinguish error types

### Estimated Effort
- Implementation: 1-2 hours
- Testing: 1 hour
- Documentation: 30 minutes

---

## Priority 3: Progress Reporting (High)

### Current State
No progress reporting for long-running operations.

### Implementation Goal
Report progress during extraction and translation for large files.

### Affected Tools
- `extract_divi_text` - Large file parsing
- `extract_wordpress_text` - Database queries
- `submit_bulk_translation` - Multi-part reassembly

### Implementation Steps

1. **Add progress notification structure**
   ```go
   type ProgressNotification struct {
       Jsonrpc string `json:"jsonrpc"`
       Method  string `json:"method"`
       Params  struct {
           ProgressToken string  `json:"progressToken"`
           Progress      int64   `json:"progress"`
           Total         int64   `json:"total"`
       } `json:"params"`
   }
   ```

2. **Send progress notifications during extraction**
   ```go
   // During tokenization loop
   if i%100 == 0 {
       s.sendProgressNotification(progressToken, int64(i), int64(totalTokens))
   }
   ```

3. **Send progress during bulk translation parsing**
   ```go
   // While parsing chunks
   for i := 0; i < totalChunks; i++ {
       s.sendProgressNotification(progressToken, int64(i), int64(totalChunks))
   }
   ```

### Implementation Code
```go
func (s *MCPServer) sendProgressNotification(token string, progress, total int64) {
    notification := map[string]interface{}{
        "jsonrpc": "2.0",
        "method":  "notifications/progress",
        "params": map[string]interface{}{
            "progressToken": token,
            "progress":      progress,
            "total":         total,
        },
    }
    data, _ := json.Marshal(notification)
    fmt.Fprintf(s.stdout, "%s\n", data)
}
```

### Files to Modify
- `mcp_server.go` (handleExtractDiviText, handleExtractWordPressText, submit handlers)
- `tokenizer.go` (optional: pass progress callback)

### Testing
- Test progress updates are sent
- Verify progress is monotonically increasing
- Verify total is accurate
- Test with different file sizes

### Estimated Effort
- Implementation: 4-6 hours
- Testing: 2 hours
- Documentation: 1 hour

---

## Priority 4: Cancellation Support (High)

### Current State
No way to cancel running operations.

### Implementation Goal
Allow clients to cancel long-running operations.

### Implementation Steps

1. **Track active operations**
   ```go
   var (
       activeOperations = make(map[string]context.Context)
       operationsMutex  sync.RWMutex
   )
   ```

2. **Add cancellation handler**
   ```go
   case "tools/cancel":
       s.handleToolCancel(req)
   ```

3. **Use context.Context for cancellation**
   ```go
   func (s *MCPServer) handleExtractDiviText(ctx context.Context, ...) {
       // Check for cancellation in loops
       select {
       case <-ctx.Done():
           return errors.New("operation cancelled")
       default:
       }
   }
   ```

4. **Return partial results on cancellation**
   ```go
   // If cancelled during extraction, return what we have
   if ctx.Err() != nil {
       s.writeResponse(JSONRPCResponse{
           Error: &RPCError{
               Code:    -32603,
               Message: "Operation cancelled by client",
           },
       })
   }
   ```

### Files to Modify
- `mcp_server.go` (all tool handlers, add context.Context)
- `tokenizer.go` (accept context, check for cancellation)
- `wordpress.go` (accept context, check for cancellation)

### Testing
- Test cancellation during large extractions
- Test partial results are saved
- Verify cleanup on cancellation
- Test rapid cancel/retry cycles

### Estimated Effort
- Implementation: 3-4 hours
- Testing: 2-3 hours
- Documentation: 1 hour

---

## Priority 5: Tool Annotations (Low)

### Current State
Tools lack UI hints and progressive disclosure.

### Implementation Goal
Add `_annotations` field to tools for better client UX.

### Implementation Example
```go
type Tool struct {
    Name         string                 `json:"name"`
    Description  string                 `json:"description"`
    InputSchema  map[string]interface{} `json:"inputSchema"`
    Annotations  *ToolAnnotations       `json:"_annotations,omitempty"`
}

type ToolAnnotations struct {
    Suggested bool   `json:"suggested,omitempty"`
    Category  string `json:"category,omitempty"`
    Urgency   string `json:"urgency,omitempty"`
}
```

### Annotations to Add

**extract_divi_text**
```json
{
    "_annotations": {
        "suggested": true,
        "category": "translation"
    }
}
```

**extract_wordpress_text**
```json
{
    "_annotations": {
        "suggested": true,
        "category": "translation"
    }
}
```

**submit_bulk_translation**
```json
{
    "_annotations": {
        "suggested": false,
        "category": "translation",
        "urgency": "required"
    }
}
```

### Files to Modify
- `mcp_server.go` (Tool struct, handleListTools)

### Testing
- Verify annotations are returned in tools/list
- Test client rendering with annotations
- Verify backward compatibility (omitted annotations)

### Estimated Effort
- Implementation: 1 hour
- Testing: 30 minutes
- Documentation: 30 minutes

---

## Priority 6: Logging Enhancements (Low)

### Current State
Basic logging to stderr.

### Implementation Goal
Structured logging with levels and timestamps.

### Changes
```go
type LogLevel string

const (
    LogDebug LogLevel = "DEBUG"
    LogInfo  LogLevel = "INFO"
    LogWarn  LogLevel = "WARN"
    LogError LogLevel = "ERROR"
)

func (s *MCPServer) logf(level LogLevel, format string, args ...interface{}) {
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    fmt.Fprintf(s.stderr, "[%s] %s: %s\n", timestamp, level, fmt.Sprintf(format, args...))
}
```

### Files to Modify
- `mcp_server.go` (log function)

### Estimated Effort
- Implementation: 1 hour
- Testing: 30 minutes

---

## Priority 7: Resource Features (Low)

### Current State
Not implemented.

### Implementation Goal
Optional: Allow clients to browse available Divi pages/posts.

### Possible Resources
```
divi://local/files           // List available files
divi://wordpress/posts       // List WordPress posts
```

### Effort Assessment
- **High effort** (6-8 hours)
- **Medium impact** (useful but not critical)
- **Recommend**: Only if users request this feature

### Files to Modify
- `mcp_server.go` (add resources/list, resources/read)
- `wordpress.go` (add listing methods)

---

## Priority 8: Prompts Feature (Very Low)

### Current State
Not implemented.

### Implementation Goal
Optional: Provide translation templates/prompts.

### Example Prompts
```
"translate-divi-file"
"translate-wordpress-post"
"fix-translation"
```

### Effort Assessment
- **Medium effort** (4-5 hours)
- **Low impact** (convenience feature)
- **Recommend**: Only if requested by users

---

## Implementation Timeline Recommendation

```
Phase 1 (v4.3.1) - Weeks 1-2
└─ Capabilities Negotiation
└─ Standard Error Codes
└─ Testing & Documentation

Phase 2 (v4.4.0) - Weeks 3-6
└─ Progress Reporting
└─ Cancellation Support
└─ Tool Annotations
└─ Logging Enhancements

Phase 3 (v4.5.0) - Future
└─ Resource Features (if requested)
└─ Prompts Feature (if requested)
└─ Additional client features
```

---

## Testing Strategy

### Unit Tests
- Create `mcp_server_test.go`
- Test each new handler independently
- Test error codes
- Test progress notifications

### Integration Tests
- Test full workflows with new features
- Test backward compatibility
- Test error recovery

### Specification Compliance
- Use MCP protocol validators
- Test with Claude Desktop
- Verify all spec requirements met

---

## Documentation Updates

1. **Update CLAUDE.md**
   - Document new capabilities
   - Add examples of progress monitoring
   - Document cancellation behavior

2. **Update README.md**
   - Add feature list
   - Document new MCP 2025-11-25 features
   - Add troubleshooting section

3. **Create IMPLEMENTATION_NOTES.md**
   - Internal implementation details
   - Architecture decisions
   - Performance considerations

---

## Conclusion

All recommended improvements maintain **backward compatibility** while adding powerful new capabilities. Implementation should be phased based on:

1. **User demand**
2. **Resource availability**
3. **Specification updates**

The server is now MCP 2025-11-25 compliant and ready for production use. These improvements are **optional enhancements** that will make the server even more robust and user-friendly.

---

**Document Version**: 1.0
**Created**: 2025-02-18
**Review Frequency**: Quarterly or as spec updates arrive
