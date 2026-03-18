# Future Improvements Roadmap
## MCP Divi Translator

**Status**: All practical improvements implemented
**Last Updated**: 2026-03-18

---

## Completed

### Standard Error Codes (v4.4.0)
- Defined JSON-RPC 2.0 error code constants (`ErrParseError`, `ErrInvalidRequest`, `ErrMethodNotFound`, `ErrInvalidParams`, `ErrInternalError`)
- Centralized error response helper `rpcError()` replacing all hardcoded error construction
- All 6 error sites in `mcp_server.go` and `handlers.go` migrated to use constants + helper

---

## Evaluated and Dismissed

The following items were evaluated and deemed unnecessary for this server's use case:

| Item | Reason for Dismissal |
|------|---------------------|
| **Capabilities Negotiation** | Server only offers `tools`; all known clients (Claude Desktop, Claude Code) support tools. No degradation path needed. |
| **Progress Reporting** | Server operations (file read, SQL query) complete in milliseconds. Translation is done by the LLM, not the server. `get_translation_status` covers polling. |
| **Cancellation Support** | No long-running server operations to cancel. Adding `context.Context` everywhere would be dead code. |
| **Tool Annotations** | No MCP clients currently consume `_annotations`. |
| **Logging Enhancements** | Current `[MCP]` prefix logging is adequate. Server runs per-session, no need for structured log levels. |
| **Resource Features** | File listing and post browsing are LLM capabilities (filesystem access, SQL). Duplicating in the server adds no value. |
| **Prompts Feature** | CLAUDE.md already provides all workflow prompts. Server-side prompts would be redundant. |

---

**Document Version**: 2.0
**Created**: 2025-02-18
**Updated**: 2026-03-18
