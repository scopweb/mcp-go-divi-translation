# Security Policy

## Reporting a Vulnerability

If you find a security vulnerability in this project, please report it responsibly:

1. **Do NOT open a public issue**
2. Use [GitHub Private Vulnerability Reporting](https://github.com/scopweb/mcp-go-divi-translation/security/advisories/new)
3. Or email the maintainer directly

## What to Report

- Credential exposure in code or config
- SQL injection in WordPress queries
- Path traversal in file operations
- Any way to execute arbitrary code through the MCP server

## Scope

This server:
- Reads and writes files only at paths you explicitly provide
- Connects to MySQL only if you configure credentials
- Has no network access (stdio transport only)
- Does not execute translated content as code
