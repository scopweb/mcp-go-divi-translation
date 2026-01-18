package main

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file from the same directory as the executable
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	envPath := filepath.Join(exeDir, ".env")

	// Try to load .env (ignore error if not exists)
	godotenv.Load(envPath)

	// Also try current working directory
	godotenv.Load(".env")

	// Run MCP server via stdio
	server := NewMCPServer()
	server.Run()
	os.Exit(0)
}
