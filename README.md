# MCP Divi Translator v4.4.0

Translates Divi pages preserving all `[et_*]` shortcodes and HTML structure. Works with Claude Desktop, Claude Code, or any MCP client.

## Install

```bash
git clone https://github.com/scopweb/mcp-go-divi-translation.git
cd mcp-go-divi-translation
go build -o divi-translator .
```

## Configure

**Claude Desktop** — edit `claude_desktop_config.json`:

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

**Claude Code** — add `.mcp.json` in your project:

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

### WordPress (optional)

Create a `.env` file next to the binary:

```ini
WP_MYSQL_HOST=localhost
WP_MYSQL_PORT=3306
WP_MYSQL_USER=your_user
WP_MYSQL_PASSWORD=your_password
WP_MYSQL_DATABASE=your_database
WP_TABLE_PREFIX=wp_
WP_BACKUP_DIR=./backups
```

## Usage

The server provides 3 tools for the recommended bulk workflow:

| Tool | What it does |
|------|-------------|
| `extract_divi_text` | Extracts text from a Divi file |
| `extract_wordpress_text` | Extracts text from a WordPress post |
| `submit_bulk_translation` | Sends the translation back |

**How it works:**

1. Call `extract_divi_text` or `extract_wordpress_text` — returns an `extractionId` and text with `{{CHUNK_XXX}}` markers
2. Translate the text (Claude does this, no tool calls needed)
3. Call `submit_bulk_translation` with the `extractionId` and translated text

Example prompt:

```
Translate WordPress post 42 to French.
Use extract_wordpress_text, translate keeping all markers, then submit with submit_bulk_translation.
```

## What gets translated

- Visible text between HTML tags
- `alt` and `title` attributes

## What stays untouched

- Divi shortcodes (`[et_pb_section]`, `[et_pb_text]`, etc.)
- HTML attributes (`class`, `style`, `href`, `src`, etc.)
- URLs
- Chunk markers (`{{CHUNK_XXX}}`)

## Documentation

Full docs at [website/](website/) (Astro Starlight site).

## License

MIT
