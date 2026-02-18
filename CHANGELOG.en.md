# Changelog

All notable changes to this project are documented in this file.

## [4.3.0] - 2025-02-18

### Added

- **server_info**: New tool that returns server information
  - Server version and MCP protocol
  - MySQL connection status
  - Active configuration (masked)
  - Active sessions
  - List of available tools

### Fixed

- Claude Desktop compatibility: Removed `_meta` fields from JSON-RPC responses that caused validation errors
- Simplified response structure to comply with Claude Desktop's strict validation

---

## [4.2.0] - 2025-01-18

### Added

- **WordPress metadata translation**: Now translates post title, slug and excerpt
  - `{{POST_TITLE}}`: Post title
  - `{{POST_SLUG}}`: Friendly URL (slug)
  - `{{POST_EXCERPT}}`: Post excerpt/summary

- **Full backup**: Backup now includes all fields (title, slug, excerpt, content)

- **UpdatePostFull**: New function that updates all post fields in a single operation

### Changed

- `extract_wordpress_text` now includes metadata section at the beginning of extracted text
- `submit_bulk_translation` parses metadata markers and updates all fields
- `SaveFullBackup` replaces partial backup to include all original data

### Example output

```
POST METADATA (translate too):
======================================
{{POST_TITLE}}
Original Title
{{/POST_TITLE}}

{{POST_SLUG}}
original-slug
{{/POST_SLUG}}

{{POST_EXCERPT}}
Original excerpt text.
{{/POST_EXCERPT}}

CONTENT TO TRANSLATE:
=====================
{{CHUNK_001}}...
```

---

## [4.1.0] - 2025-01-18

### Added

- **extractionId**: Each extraction now has a unique ID to identify the session
- **Multi-session support**: Multiple extractions can run in parallel
- **Global storage**: Sessions are stored in memory with mutex for thread-safety

### Changed

- `extract_divi_text` and `extract_wordpress_text` now return `extractionId` in response
- `submit_bulk_translation` now requires `extractionId` as mandatory parameter
- Response messages include `extractionId` for easy tracking

### Benefits

- Claude Desktop can reference the session by ID instead of relying on server state
- More robust against disconnections or errors
- Prepared for parallel translations

---

## [4.0.0] - 2025-01-18

### Added

- **BULK mode (Optimized)**: New translation flow that drastically reduces MCP calls
  - `extract_divi_text`: Extracts ALL text from a Divi file in a single document with `{{CHUNK_XXX}}` markers
  - `extract_wordpress_text`: Same but from WordPress MySQL
  - `submit_bulk_translation`: Receives complete translated text, parses markers and reassembles the document

- **Automatic partitioning**: If text exceeds 30KB, it's automatically divided into 2-3 parts

- **Chunk markers**: `{{CHUNK_001}}...{{/CHUNK_001}}` marker system to identify text blocks

### Changed

- Optimized translation flow: from N+1 MCP calls to just 2 calls
- Updated README.md with BULK mode documentation
- Updated CLAUDE.md with instructions for Claude Desktop

### Why this change?

The legacy mode (chunk-by-chunk) required one MCP call per text block. In large documents (60+ chunks), Claude Desktop would lose context and fail before completing the translation.

The new BULK mode:
1. Extracts all text in a single call
2. Claude translates without calling tools (minimal token consumption)
3. Reassembles in a single call

**Result**: Translations that previously failed now complete successfully.

---

## [3.0.0] - 2025-01-17

### Added

- Support for direct WordPress MySQL
  - `start_wordpress_translation`: Reads posts directly from database
  - Automatic backup of original content
  - Automatic database update upon completion

- Environment variables for MySQL configuration:
  - `WP_MYSQL_HOST`, `WP_MYSQL_PORT`, `WP_MYSQL_USER`
  - `WP_MYSQL_PASSWORD`, `WP_MYSQL_DATABASE`
  - `WP_TABLE_PREFIX`, `WP_BACKUP_DIR`

### Changed

- Refactored session system to support multiple sources

---

## [2.0.0] - 2025-01-16

### Added

- Improved tokenization system for Divi shortcodes
- Support for closing shortcodes `[/et_*]`
- Automatic cleanup of empty HTML tags

### Changed

- Better handling of special characters in attributes

---

## [1.0.0] - 2025-01-15

### Added

- Initial MCP server implementation
- `start_divi_translation`: Starts translation from file
- `submit_translation`: Submits translation of a chunk
- `get_translation_status`: Shows progress
- Basic tokenizer for `[et_*]` shortcodes
