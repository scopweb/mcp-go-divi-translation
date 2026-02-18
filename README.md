# MCP Divi Translator v4.3.0

[![MCP Spec](https://img.shields.io/badge/MCP-2025--11--25-blue)](https://modelcontextprotocol.io/specification/2025-11-25)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

> **Automatic Divi page translator for Claude Desktop** that preserves Divi shortcodes `[et_*]` and HTML structure during translation.

## 🎯 Features

- **Bulk Translation Mode** - Extract, translate, and reassemble in 2 MCP calls
- **Legacy Chunk-by-Chunk** - Fallback mode for compatibility
- **File-Based** - Translate local `.txt` files
- **WordPress Integration** - Direct database support with automatic backups
- **Shortcode Preservation** - Divi shortcodes remain untouched
- **HTML Structure Preservation** - All HTML tags and attributes protected
- **Multi-Platform** - Windows, macOS (arm64), Linux
- **MCP 2025-11-25 Compliant** - Full specification compliance

## 📋 Requirements

- **Go** 1.21 or higher
- **Claude Desktop** with MCP support
- **MySQL** (optional, only for WordPress mode)

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/scp-divi-translation.git
cd scp-divi-translation

# Build the server
go mod tidy
go build -o divi-translator .
```

### Claude Desktop Configuration

#### File-Only Mode

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or equivalent:

```json
{
  "mcpServers": {
    "divi-translator": {
      "command": "/path/to/divi-translator",
      "args": []
    }
  }
}
```

#### With WordPress Database Support

```json
{
  "mcpServers": {
    "divi-translator": {
      "command": "/path/to/divi-translator",
      "args": [],
      "env": {
        "WP_MYSQL_HOST": "your-mysql-server.com",
        "WP_MYSQL_PORT": "3306",
        "WP_MYSQL_USER": "your_username",
        "WP_MYSQL_PASSWORD": "your_password",
        "WP_MYSQL_DATABASE": "your_database",
        "WP_TABLE_PREFIX": "wp_",
        "WP_BACKUP_DIR": "/path/to/backups"
      }
    }
  }
}
```

## 📚 Available Tools

### BULK Mode (Recommended ⭐)

| Tool | Purpose |
|------|---------|
| `extract_divi_text` | Extract all text from a Divi file with chunk markers |
| `extract_wordpress_text` | Extract all text from a WordPress post with metadata |
| `submit_bulk_translation` | Submit complete translated text and reassemble |

**Usage Pattern:**
1. Call `extract_divi_text` or `extract_wordpress_text`
2. Claude translates the text (no tool calls needed)
3. Call `submit_bulk_translation` with `extractionId`

### Legacy Mode (Fallback)

| Tool | Purpose |
|------|---------|
| `start_divi_translation` | Start chunk-by-chunk translation from file |
| `start_wordpress_translation` | Start chunk-by-chunk translation from WordPress |
| `submit_translation` | Submit translated chunk and get next one |

### Utilities

| Tool | Purpose |
|------|---------|
| `get_translation_status` | Show current translation progress |

## 💬 Usage in Claude Desktop

### Translate a Divi File to Spanish

```
Translate this Divi page to Spanish:
- inputPath: /path/to/page.txt
- outputPath: /path/to/page.es.txt
- targetLang: es

Use extract_divi_text to extract the text, translate all text while preserving {{CHUNK_XXX}} markers,
then use submit_bulk_translation with the extractionId and translated text.
```

### Translate a WordPress Post

```
Translate WordPress post to French:
- postId: 123
- targetLang: fr

Use extract_wordpress_text to extract, translate while preserving all markers
({{POST_TITLE}}, {{POST_SLUG}}, {{POST_EXCERPT}}, {{CHUNK_XXX}}),
then submit with submit_bulk_translation.
```

## 📋 Translation Rules

### DO Translate ✓
- Visible text content
- Image `alt` attributes
- Element `title` attributes

### DON'T Translate ✗
- Divi shortcodes: `[et_pb_section]`, `[et_pb_text]`, etc.
- Shortcode closing tags: `[/et_pb_section]`, etc.
- Shortcode attributes: `_builder_version`, `global_colors_info`, etc.
- HTML attributes: `class`, `style`, `href`, `src`, `id`, `data-*`, `width`, `height`
- Complete URLs
- Chunk markers: `{{CHUNK_XXX}}`, `{{/CHUNK_XXX}}`
- Metadata markers: `{{POST_TITLE}}`, `{{POST_SLUG}}`, `{{POST_EXCERPT}}`

### Preserve Structure
- Exact HTML structure
- Line breaks and whitespace
- HTML entities (`&nbsp;`, `&amp;`, etc.)
- Remove only empty tags: `<p></p>`, `<span></span>`

## 🔧 Environment Variables

### WordPress Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `WP_MYSQL_HOST` | MySQL server hostname | `localhost` |
| `WP_MYSQL_PORT` | MySQL server port | `3306` |
| `WP_MYSQL_USER` | MySQL username | (required) |
| `WP_MYSQL_PASSWORD` | MySQL password | (empty) |
| `WP_MYSQL_DATABASE` | Database name | (required) |
| `WP_TABLE_PREFIX` | WordPress table prefix | `wp_` |
| `WP_BACKUP_DIR` | Backup directory path | `.` |

## 📁 Chunk Format

The server generates text with markers that must be preserved during translation:

**Input:**
```
{{CHUNK_001}}
<h2>Original Title</h2>
<p>Paragraph to translate.</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<p>More text here.</p>
{{/CHUNK_002}}
```

**Output:**
```
{{CHUNK_001}}
<h2>Título Traducido</h2>
<p>Párrafo traducido.</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<p>Más texto aquí.</p>
{{/CHUNK_002}}
```

## 🔀 Automatic Partitioning

For large documents (>30KB text), the server automatically splits into 2-3 parts to avoid context limits. Each part must be translated separately, then combined automatically.

## 🎯 Language Codes

| Code | Language |
|------|----------|
| `es` | Spanish |
| `en` | English |
| `fr` | French |
| `de` | German |
| `it` | Italian |
| `pt` | Portuguese |
| `nl` | Dutch |
| `ca` | Catalan |
| `ja` | Japanese |
| `zh` | Chinese |

## 🏗️ Architecture

```
Claude Desktop (MCP Client)
         │
         ├─ extract_divi_text ──┐
         │                      ├─→ Tokenization
         └─ submit_bulk_translation ──┤
                                      ├─→ Chunk parsing
                                      ├─→ HTML reassembly
                                      └─→ File/DB update
```

### Key Components

- **tokenizer.go** - HTML/shortcode tokenization
- **mcp_server.go** - MCP protocol implementation
- **wordpress.go** - WordPress database integration
- **main.go** - Server entry point

## 📊 Performance

- **Extraction**: < 1s for typical pages
- **Reassembly**: < 500ms for typical pages
- **Large Files**: Automatic partitioning (2-3 parts)
- **Database Operations**: Optimized with backup support

## 🔐 Security

- ✓ No credentials in code
- ✓ `.env` file support for database credentials
- ✓ Automatic backups before updates
- ✓ HTML sanitization (empty tags removed)
- ✓ No code execution on translated content

## 📝 Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

### v4.3.0 (2025-02-18)
- **NEW**: Full MCP 2025-11-25 specification compliance
- **NEW**: Ping/keepalive support
- **NEW**: Proper shutdown handling
- **FIXED**: Protocol version updated
- **IMPROVED**: Request ID validation
- **IMPROVED**: Response metadata (`_meta` field)

## 🛠️ Development

### Build

```bash
go build -o divi-translator .
```

### Test

```bash
go test ./...
```

### Run

```bash
./divi-translator
```

## 📖 Documentation

- **[AUDIT_REPORT_2025-02-18.md](AUDIT_REPORT_2025-02-18.md)** - MCP compliance audit
- **[SPEC_UPDATE_SUMMARY.md](SPEC_UPDATE_SUMMARY.md)** - Implementation details
- **[CLAUDE.md](CLAUDE.md)** - Claude Desktop integration guide
- **[FUTURE_IMPROVEMENTS.md](FUTURE_IMPROVEMENTS.md)** - Roadmap

## 📦 Installation Methods

### macOS (Homebrew)

```bash
# If published to Homebrew
brew install divi-translator
```

### Docker

```bash
docker pull yourusername/divi-translator
docker run -e WP_MYSQL_HOST=host.docker.internal yourusername/divi-translator
```

### Linux

```bash
curl -L https://github.com/yourusername/scp-divi-translation/releases/download/v4.3.0/divi-translator-linux-x64 -o divi-translator
chmod +x divi-translator
./divi-translator
```

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push and create a Pull Request

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙋 Support

For issues, questions, or feature requests:
- Open an [GitHub Issue](https://github.com/yourusername/scp-divi-translation/issues)
- Check [existing documentation](./docs)
- Review [MCP specification](https://modelcontextprotocol.io/)

## 🔗 Resources

- [MCP Protocol Specification](https://modelcontextprotocol.io/specification/2025-11-25)
- [Claude Desktop Documentation](https://claude.ai/resources/docs)
- [Divi Builder Documentation](https://www.elegantthemes.com/gallery/divi/)
- [Go Documentation](https://golang.org/doc)

---

**Made with ❤️ for translating Divi pages automatically**

**Version**: 4.3.0 | **MCP Spec**: 2025-11-25 | **Status**: Production Ready
