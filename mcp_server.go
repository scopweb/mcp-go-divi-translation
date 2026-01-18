package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// MCP JSON-RPC types
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
		Tools map[string]interface{} `json:"tools,omitempty"`
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
	// Source: file or wordpress
	SourceType   string // "file" or "wordpress"
	InputPath    string // For file source
	OutputPath   string // For file source
	PostID       int64  // For wordpress source
	BackupPath   string // For wordpress source
	TargetLang   string
	Tokens       []Token
	TextChunks   []string // Solo los textos a traducir
	ChunkIndices []int    // Indices de tokens que son texto
	Translations []string // Traducciones recibidas
	CurrentChunk int      // Chunk actual a traducir
	TotalChunks  int
}

// BulkTranslationSession holds state for the optimized bulk translation flow
type BulkTranslationSession struct {
	SourceType   string   // "file" or "wordpress"
	InputPath    string   // For file source
	OutputPath   string   // For file source
	PostID       int64    // For wordpress source
	BackupPath   string   // For wordpress source
	TargetLang   string
	Tokens       []Token
	ChunkIndices []int    // Indices of text tokens
	TotalChunks  int
	Parts        int      // Number of parts (1, 2, or 3)
	CurrentPart  int      // Current part being translated
	PartRanges   [][2]int // Start/end indices for each part
	Translations []string // Collected translations per chunk
}

// MCPServer implements the MCP protocol
type MCPServer struct {
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	session     *TranslationSession     // Estado de la sesion actual (legacy)
	bulkSession *BulkTranslationSession // Estado de la sesion bulk (optimizado)
	wpDB        *WordPressDB            // Conexion WordPress (lazy init)
}

func NewMCPServer() *MCPServer {
	return &MCPServer{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

func (s *MCPServer) log(format string, args ...interface{}) {
	fmt.Fprintf(s.stderr, "[MCP] "+format+"\n", args...)
}

func (s *MCPServer) writeResponse(resp JSONRPCResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.stdout, "%s\n", data)
	return err
}

// getWordPressDB returns the WordPress DB connection, initializing if needed
func (s *MCPServer) getWordPressDB() (*WordPressDB, error) {
	if s.wpDB != nil {
		return s.wpDB, nil
	}

	wpDB, err := NewWordPressDB()
	if err != nil {
		return nil, err
	}
	s.wpDB = wpDB
	return s.wpDB, nil
}

func (s *MCPServer) handleInitialize(req JSONRPCRequest) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
	}
	result.Capabilities.Tools = map[string]interface{}{}
	result.ServerInfo.Name = "divi-translator"
	result.ServerInfo.Version = "3.0.0"

	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	})
}

func (s *MCPServer) handleListTools(req JSONRPCRequest) {
	tools := []Tool{
		// File-based translation
		{
			Name:        "start_divi_translation",
			Description: "Inicia la traduccion de una pagina Divi DESDE ARCHIVO. Devuelve el primer chunk de texto a traducir.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"inputPath": map[string]interface{}{
						"type":        "string",
						"description": "Ruta absoluta del fichero Divi a traducir",
					},
					"outputPath": map[string]interface{}{
						"type":        "string",
						"description": "Ruta donde guardar el fichero traducido",
					},
					"targetLang": map[string]interface{}{
						"type":        "string",
						"description": "Codigo de idioma destino (es, en, fr, de, it, pt, etc.)",
					},
				},
				"required": []string{"inputPath", "outputPath", "targetLang"},
			},
		},
		// WordPress-based translation
		{
			Name:        "start_wordpress_translation",
			Description: "Inicia la traduccion de un post de WordPress DESDE BASE DE DATOS. Lee el post, crea backup y devuelve el primer chunk. Al finalizar actualiza automaticamente la BD.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"postId": map[string]interface{}{
						"type":        "integer",
						"description": "ID del post de WordPress a traducir",
					},
					"targetLang": map[string]interface{}{
						"type":        "string",
						"description": "Codigo de idioma destino (es, en, fr, de, it, pt, etc.)",
					},
				},
				"required": []string{"postId", "targetLang"},
			},
		},
		// Common submit tool
		{
			Name:        "submit_translation",
			Description: "Envia la traduccion del chunk actual. Devuelve el siguiente chunk, o confirma que se guardo (archivo o BD) si era el ultimo.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"translatedText": map[string]interface{}{
						"type":        "string",
						"description": "El texto traducido del chunk actual (solo el texto, sin marcadores)",
					},
				},
				"required": []string{"translatedText"},
			},
		},
		// Status tool
		{
			Name:        "get_translation_status",
			Description: "Obtiene el estado actual de la traduccion en progreso.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		// ============ BULK TRANSLATION (OPTIMIZED) ============
		// These tools minimize MCP calls: extract once, translate text, reassemble once
		{
			Name:        "extract_divi_text",
			Description: "OPTIMIZADO: Extrae TODO el texto traducible de un archivo Divi en UN SOLO documento. Devuelve texto con marcadores {{CHUNK_001}}...{{/CHUNK_001}} para traducir. Usa submit_bulk_translation cuando termines de traducir.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"inputPath": map[string]interface{}{
						"type":        "string",
						"description": "Ruta absoluta del archivo Divi a procesar",
					},
					"outputPath": map[string]interface{}{
						"type":        "string",
						"description": "Ruta donde guardar el archivo traducido final",
					},
					"targetLang": map[string]interface{}{
						"type":        "string",
						"description": "Codigo de idioma destino (es, en, fr, de, etc.)",
					},
				},
				"required": []string{"inputPath", "outputPath", "targetLang"},
			},
		},
		{
			Name:        "extract_wordpress_text",
			Description: "OPTIMIZADO: Extrae TODO el texto traducible de un post WordPress en UN SOLO documento. Devuelve texto con marcadores {{CHUNK_001}}...{{/CHUNK_001}} para traducir. Usa submit_bulk_translation cuando termines.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"postId": map[string]interface{}{
						"type":        "integer",
						"description": "ID del post de WordPress",
					},
					"targetLang": map[string]interface{}{
						"type":        "string",
						"description": "Codigo de idioma destino (es, en, fr, de, etc.)",
					},
				},
				"required": []string{"postId", "targetLang"},
			},
		},
		{
			Name:        "submit_bulk_translation",
			Description: "OPTIMIZADO: Recibe el texto traducido (con marcadores {{CHUNK_XXX}}), reensambla el documento Divi y lo guarda. Llamar despues de traducir el texto de extract_divi_text o extract_wordpress_text.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"translatedText": map[string]interface{}{
						"type":        "string",
						"description": "El texto traducido completo, manteniendo los marcadores {{CHUNK_XXX}}...{{/CHUNK_XXX}}",
					},
				},
				"required": []string{"translatedText"},
			},
		},
	}

	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ListToolsResult{Tools: tools},
	})
}

func (s *MCPServer) handleCallTool(req JSONRPCRequest) {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32602,
				Message: fmt.Sprintf("Invalid params: %v", err),
			},
		})
		return
	}

	switch params.Name {
	case "start_divi_translation":
		s.handleStartTranslation(req, params)
	case "start_wordpress_translation":
		s.handleStartWordPressTranslation(req, params)
	case "submit_translation":
		s.handleSubmitTranslation(req, params)
	case "get_translation_status":
		s.handleGetStatus(req)
	// Bulk translation (optimized)
	case "extract_divi_text":
		s.handleExtractDiviText(req, params)
	case "extract_wordpress_text":
		s.handleExtractWordPressText(req, params)
	case "submit_bulk_translation":
		s.handleSubmitBulkTranslation(req, params)
	default:
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Unknown tool: %s", params.Name),
			},
		})
	}
}

func (s *MCPServer) handleStartTranslation(req JSONRPCRequest, params CallToolParams) {
	inputPath, _ := params.Arguments["inputPath"].(string)
	outputPath, _ := params.Arguments["outputPath"].(string)
	targetLang, _ := params.Arguments["targetLang"].(string)

	if inputPath == "" || outputPath == "" || targetLang == "" {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "ERROR: inputPath, outputPath y targetLang son obligatorios",
				}},
				IsError: true,
			},
		})
		return
	}

	// Read file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("ERROR leyendo fichero: %v", err),
				}},
				IsError: true,
			},
		})
		return
	}

	s.initSession(string(data), targetLang, "file", inputPath, outputPath, 0, "")

	if s.session == nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "El archivo no contiene texto para traducir (solo shortcodes).",
				}},
				IsError: true,
			},
		})
		return
	}

	s.log("Sesion archivo iniciada: %d tokens, %d chunks de texto", len(s.session.Tokens), s.session.TotalChunks)

	// Return first chunk
	response := fmt.Sprintf(`TRADUCCION DESDE ARCHIVO INICIADA
==================================
Archivo: %s
Destino: %s
Idioma: %s
Total chunks de texto: %d

CHUNK 1 de %d
=============
Traduce el siguiente texto a %s.

REGLAS:
1. TRADUCIR: texto visible, atributos "title" y "alt"
2. NO TRADUCIR: class, style, href, src, id, width, height, data-*
3. Conservar estructura HTML y shortcodes [caption][/caption]
4. Eliminar etiquetas vacias (<p></p>)
5. Conservar saltos de linea

TEXTO:
%s

Cuando termines, usa "submit_translation" con el texto traducido.`,
		inputPath, outputPath, targetLang, s.session.TotalChunks,
		s.session.TotalChunks, targetLang, s.session.TextChunks[0])

	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: response,
			}},
		},
	})
}

func (s *MCPServer) handleStartWordPressTranslation(req JSONRPCRequest, params CallToolParams) {
	postIDFloat, _ := params.Arguments["postId"].(float64)
	postID := int64(postIDFloat)
	targetLang, _ := params.Arguments["targetLang"].(string)

	if postID == 0 || targetLang == "" {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "ERROR: postId y targetLang son obligatorios",
				}},
				IsError: true,
			},
		})
		return
	}

	// Get WordPress DB connection
	wpDB, err := s.getWordPressDB()
	if err != nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("ERROR conectando a WordPress: %v", err),
				}},
				IsError: true,
			},
		})
		return
	}

	// Read post and create backup
	post, backupPath, err := wpDB.ReadPostForTranslation(postID, targetLang)
	if err != nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("ERROR leyendo post: %v", err),
				}},
				IsError: true,
			},
		})
		return
	}

	s.initSession(post.PostContent, targetLang, "wordpress", "", "", postID, backupPath)

	if s.session == nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "El post no contiene texto Divi para traducir (solo shortcodes).",
				}},
				IsError: true,
			},
		})
		return
	}

	s.log("Sesion WordPress iniciada: Post %d, %d tokens, %d chunks de texto", postID, len(s.session.Tokens), s.session.TotalChunks)

	// Return first chunk
	response := fmt.Sprintf(`TRADUCCION DESDE WORDPRESS INICIADA
====================================
Post ID: %d
Titulo: %s
Tipo: %s
Estado: %s
Backup: %s
Idioma destino: %s
Total chunks de texto: %d

CHUNK 1 de %d
=============
Traduce el siguiente texto a %s.

REGLAS:
1. TRADUCIR: texto visible, atributos "title" y "alt"
2. NO TRADUCIR: class, style, href, src, id, width, height, data-*
3. Conservar estructura HTML y shortcodes [caption][/caption]
4. Eliminar etiquetas vacias (<p></p>)
5. Conservar saltos de linea

TEXTO:
%s

Cuando termines, usa "submit_translation" con el texto traducido.`,
		post.ID, post.PostTitle, post.PostType, post.PostStatus,
		backupPath, targetLang, s.session.TotalChunks,
		s.session.TotalChunks, targetLang, s.session.TextChunks[0])

	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: response,
			}},
		},
	})
}

func (s *MCPServer) initSession(content, targetLang, sourceType, inputPath, outputPath string, postID int64, backupPath string) {
	// Tokenize
	tokens := tokenize(content)

	// Extract text chunks and their indices
	var textChunks []string
	var chunkIndices []int
	for i, t := range tokens {
		if t.Kind == "text" && strings.TrimSpace(t.Value) != "" {
			textChunks = append(textChunks, t.Value)
			chunkIndices = append(chunkIndices, i)
		}
	}

	if len(textChunks) == 0 {
		s.session = nil
		return
	}

	// Initialize session
	s.session = &TranslationSession{
		SourceType:   sourceType,
		InputPath:    inputPath,
		OutputPath:   outputPath,
		PostID:       postID,
		BackupPath:   backupPath,
		TargetLang:   targetLang,
		Tokens:       tokens,
		TextChunks:   textChunks,
		ChunkIndices: chunkIndices,
		Translations: make([]string, len(textChunks)),
		CurrentChunk: 0,
		TotalChunks:  len(textChunks),
	}
}

func (s *MCPServer) handleSubmitTranslation(req JSONRPCRequest, params CallToolParams) {
	if s.session == nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "ERROR: No hay ninguna traduccion en progreso. Usa start_divi_translation o start_wordpress_translation primero.",
				}},
				IsError: true,
			},
		})
		return
	}

	translatedText, _ := params.Arguments["translatedText"].(string)
	if translatedText == "" {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "ERROR: translatedText es obligatorio",
				}},
				IsError: true,
			},
		})
		return
	}

	// Store translation
	s.session.Translations[s.session.CurrentChunk] = translatedText
	s.session.CurrentChunk++

	s.log("Chunk %d/%d traducido", s.session.CurrentChunk, s.session.TotalChunks)

	// Check if we're done
	if s.session.CurrentChunk >= s.session.TotalChunks {
		// All chunks translated, save based on source type
		var result string
		if s.session.SourceType == "wordpress" {
			result = s.saveToWordPress()
		} else {
			result = s.saveTranslatedFile()
		}
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: result,
				}},
			},
		})
		return
	}

	// Return next chunk
	nextChunk := s.session.TextChunks[s.session.CurrentChunk]
	response := fmt.Sprintf(`CHUNK %d de %d RECIBIDO

CHUNK %d de %d
=============
Traduce el siguiente texto a %s.
IMPORTANTE:
- Traduce SOLO el texto visible
- Conserva TODAS las etiquetas HTML y sus atributos
- NO traduzcas atributos de etiquetas
- Si una etiqueta queda vacia, eliminala
- Conserva saltos de linea

TEXTO A TRADUCIR:
%s

Cuando termines, usa "submit_translation" con el texto traducido.`,
		s.session.CurrentChunk, s.session.TotalChunks,
		s.session.CurrentChunk+1, s.session.TotalChunks,
		s.session.TargetLang, nextChunk)

	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: response,
			}},
		},
	})
}

func (s *MCPServer) saveTranslatedFile() string {
	// Replace text tokens with translations
	for i, idx := range s.session.ChunkIndices {
		s.session.Tokens[idx].Value = s.session.Translations[i]
	}

	// Rebuild the document
	var builder strings.Builder
	for _, t := range s.session.Tokens {
		builder.WriteString(t.Value)
	}

	result := builder.String()

	// Clean empty tags
	result = dropEmptyPTags(result)

	// Save to file
	err := os.WriteFile(s.session.OutputPath, []byte(result), 0644)
	if err != nil {
		return fmt.Sprintf("ERROR guardando archivo: %v", err)
	}

	outputPath := s.session.OutputPath
	totalChunks := s.session.TotalChunks

	// Clear session
	s.session = nil

	return fmt.Sprintf(`TRADUCCION COMPLETADA (ARCHIVO)
===============================
Archivo guardado: %s
Chunks traducidos: %d

El archivo Divi ha sido traducido exitosamente.
Los shortcodes [et_*] se han preservado intactos.
Las etiquetas HTML vacias han sido eliminadas.`, outputPath, totalChunks)
}

func (s *MCPServer) saveToWordPress() string {
	// Replace text tokens with translations
	for i, idx := range s.session.ChunkIndices {
		s.session.Tokens[idx].Value = s.session.Translations[i]
	}

	// Rebuild the document
	var builder strings.Builder
	for _, t := range s.session.Tokens {
		builder.WriteString(t.Value)
	}

	result := builder.String()

	// Clean empty tags
	result = dropEmptyPTags(result)

	// Update WordPress
	wpDB, err := s.getWordPressDB()
	if err != nil {
		return fmt.Sprintf("ERROR conectando a WordPress: %v", err)
	}

	err = wpDB.UpdatePostContent(s.session.PostID, result)
	if err != nil {
		return fmt.Sprintf("ERROR actualizando post: %v", err)
	}

	postID := s.session.PostID
	backupPath := s.session.BackupPath
	totalChunks := s.session.TotalChunks

	// Clear session
	s.session = nil

	return fmt.Sprintf(`TRADUCCION COMPLETADA (WORDPRESS)
=================================
Post ID actualizado: %d
Backup original: %s
Chunks traducidos: %d

El post de WordPress ha sido actualizado exitosamente.
Los shortcodes [et_*] se han preservado intactos.
Las etiquetas HTML vacias han sido eliminadas.

IMPORTANTE: El backup del contenido original esta en:
%s`, postID, backupPath, totalChunks, backupPath)
}

// ============ BULK TRANSLATION HANDLERS (OPTIMIZED) ============

const maxCharsPerPart = 30000 // ~30KB per part, safe for Claude context

func (s *MCPServer) handleExtractDiviText(req JSONRPCRequest, params CallToolParams) {
	inputPath, _ := params.Arguments["inputPath"].(string)
	outputPath, _ := params.Arguments["outputPath"].(string)
	targetLang, _ := params.Arguments["targetLang"].(string)

	if inputPath == "" || outputPath == "" || targetLang == "" {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "ERROR: inputPath, outputPath y targetLang son obligatorios",
				}},
				IsError: true,
			},
		})
		return
	}

	// Read file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("ERROR leyendo archivo: %v", err),
				}},
				IsError: true,
			},
		})
		return
	}

	s.initBulkSession(string(data), targetLang, "file", inputPath, outputPath, 0, "")

	if s.bulkSession == nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "El archivo no contiene texto para traducir (solo shortcodes).",
				}},
				IsError: true,
			},
		})
		return
	}

	s.log("Sesion bulk archivo iniciada: %d chunks, %d partes", s.bulkSession.TotalChunks, s.bulkSession.Parts)

	// Generate and return first part
	response := s.generateBulkExtractResponse()
	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: response,
			}},
		},
	})
}

func (s *MCPServer) handleExtractWordPressText(req JSONRPCRequest, params CallToolParams) {
	postIDFloat, _ := params.Arguments["postId"].(float64)
	postID := int64(postIDFloat)
	targetLang, _ := params.Arguments["targetLang"].(string)

	if postID == 0 || targetLang == "" {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "ERROR: postId y targetLang son obligatorios",
				}},
				IsError: true,
			},
		})
		return
	}

	// Get WordPress DB connection
	wpDB, err := s.getWordPressDB()
	if err != nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("ERROR conectando a WordPress: %v", err),
				}},
				IsError: true,
			},
		})
		return
	}

	// Read post and create backup
	post, backupPath, err := wpDB.ReadPostForTranslation(postID, targetLang)
	if err != nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("ERROR leyendo post: %v", err),
				}},
				IsError: true,
			},
		})
		return
	}

	s.initBulkSession(post.PostContent, targetLang, "wordpress", "", "", postID, backupPath)

	if s.bulkSession == nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "El post no contiene texto Divi para traducir.",
				}},
				IsError: true,
			},
		})
		return
	}

	s.log("Sesion bulk WordPress iniciada: Post %d, %d chunks, %d partes", postID, s.bulkSession.TotalChunks, s.bulkSession.Parts)

	// Generate and return first part
	response := s.generateBulkExtractResponse()
	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: response,
			}},
		},
	})
}

func (s *MCPServer) initBulkSession(content, targetLang, sourceType, inputPath, outputPath string, postID int64, backupPath string) {
	// Tokenize
	tokens := tokenize(content)

	// Extract text chunk indices
	var chunkIndices []int
	for i, t := range tokens {
		if t.Kind == "text" && strings.TrimSpace(t.Value) != "" {
			chunkIndices = append(chunkIndices, i)
		}
	}

	if len(chunkIndices) == 0 {
		s.bulkSession = nil
		return
	}

	// Calculate total text length to determine parts
	totalLen := 0
	for _, idx := range chunkIndices {
		totalLen += len(tokens[idx].Value)
	}

	// Determine number of parts (1, 2, or 3)
	parts := 1
	if totalLen > maxCharsPerPart*2 {
		parts = 3
	} else if totalLen > maxCharsPerPart {
		parts = 2
	}

	// Calculate part ranges (distribute chunks evenly)
	partRanges := make([][2]int, parts)
	chunksPerPart := (len(chunkIndices) + parts - 1) / parts
	for p := 0; p < parts; p++ {
		start := p * chunksPerPart
		end := start + chunksPerPart
		if end > len(chunkIndices) {
			end = len(chunkIndices)
		}
		partRanges[p] = [2]int{start, end}
	}

	s.bulkSession = &BulkTranslationSession{
		SourceType:   sourceType,
		InputPath:    inputPath,
		OutputPath:   outputPath,
		PostID:       postID,
		BackupPath:   backupPath,
		TargetLang:   targetLang,
		Tokens:       tokens,
		ChunkIndices: chunkIndices,
		TotalChunks:  len(chunkIndices),
		Parts:        parts,
		CurrentPart:  0,
		PartRanges:   partRanges,
		Translations: make([]string, len(chunkIndices)),
	}
}

func (s *MCPServer) generateBulkExtractResponse() string {
	session := s.bulkSession
	partRange := session.PartRanges[session.CurrentPart]

	var builder strings.Builder

	// Header
	if session.Parts == 1 {
		builder.WriteString(fmt.Sprintf(`EXTRACCION COMPLETADA - TEXTO PARA TRADUCIR
============================================
Origen: %s
Idioma destino: %s
Total de bloques: %d

INSTRUCCIONES:
1. Traduce TODO el texto a %s
2. CONSERVA los marcadores {{CHUNK_XXX}} y {{/CHUNK_XXX}} exactamente igual
3. NO traduzcas atributos HTML (class, style, href, src, id, data-*)
4. SI traduce atributos "title" y "alt"
5. Conserva la estructura HTML y saltos de linea
6. Cuando termines, usa "submit_bulk_translation" con el texto traducido

TEXTO A TRADUCIR:
=================
`, s.getSourceDescription(), session.TargetLang, session.TotalChunks, session.TargetLang))
	} else {
		builder.WriteString(fmt.Sprintf(`EXTRACCION COMPLETADA - PARTE %d de %d
======================================
Origen: %s
Idioma destino: %s
Bloques en esta parte: %d-%d de %d total

INSTRUCCIONES:
1. Traduce TODO el texto a %s
2. CONSERVA los marcadores {{CHUNK_XXX}} y {{/CHUNK_XXX}} exactamente igual
3. NO traduzcas atributos HTML (class, style, href, src, id, data-*)
4. SI traduce atributos "title" y "alt"
5. Conserva la estructura HTML y saltos de linea
6. Cuando termines, usa "submit_bulk_translation" con el texto traducido

TEXTO A TRADUCIR (PARTE %d):
============================
`, session.CurrentPart+1, session.Parts, s.getSourceDescription(), session.TargetLang,
			partRange[0]+1, partRange[1], session.TotalChunks, session.TargetLang, session.CurrentPart+1))
	}

	// Generate text blocks with markers
	for i := partRange[0]; i < partRange[1]; i++ {
		chunkIdx := session.ChunkIndices[i]
		text := session.Tokens[chunkIdx].Value
		builder.WriteString(fmt.Sprintf("\n{{CHUNK_%03d}}\n%s\n{{/CHUNK_%03d}}\n", i+1, text, i+1))
	}

	return builder.String()
}

func (s *MCPServer) getSourceDescription() string {
	if s.bulkSession.SourceType == "wordpress" {
		return fmt.Sprintf("WordPress Post ID %d", s.bulkSession.PostID)
	}
	return s.bulkSession.InputPath
}

func (s *MCPServer) handleSubmitBulkTranslation(req JSONRPCRequest, params CallToolParams) {
	if s.bulkSession == nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "ERROR: No hay sesion de extraccion activa. Usa extract_divi_text o extract_wordpress_text primero.",
				}},
				IsError: true,
			},
		})
		return
	}

	translatedText, _ := params.Arguments["translatedText"].(string)
	if translatedText == "" {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "ERROR: translatedText es obligatorio",
				}},
				IsError: true,
			},
		})
		return
	}

	// Parse translated chunks from the text
	err := s.parseBulkTranslation(translatedText)
	if err != nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("ERROR parseando traduccion: %v", err),
				}},
				IsError: true,
			},
		})
		return
	}

	s.bulkSession.CurrentPart++

	// Check if there are more parts
	if s.bulkSession.CurrentPart < s.bulkSession.Parts {
		// Return next part
		response := s.generateBulkExtractResponse()
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("PARTE %d RECIBIDA\n\n%s", s.bulkSession.CurrentPart, response),
				}},
			},
		})
		return
	}

	// All parts received, save the result
	var result string
	if s.bulkSession.SourceType == "wordpress" {
		result = s.saveBulkToWordPress()
	} else {
		result = s.saveBulkToFile()
	}

	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: result,
			}},
		},
	})
}

func (s *MCPServer) parseBulkTranslation(text string) error {
	session := s.bulkSession
	partRange := session.PartRanges[session.CurrentPart]

	// Parse each chunk marker
	for i := partRange[0]; i < partRange[1]; i++ {
		marker := fmt.Sprintf("{{CHUNK_%03d}}", i+1)
		endMarker := fmt.Sprintf("{{/CHUNK_%03d}}", i+1)

		startIdx := strings.Index(text, marker)
		if startIdx == -1 {
			return fmt.Errorf("marcador %s no encontrado en la traduccion", marker)
		}

		endIdx := strings.Index(text, endMarker)
		if endIdx == -1 {
			return fmt.Errorf("marcador de cierre %s no encontrado", endMarker)
		}

		// Extract translated content between markers
		contentStart := startIdx + len(marker)
		translated := strings.TrimSpace(text[contentStart:endIdx])

		// Preserve original leading/trailing whitespace pattern
		original := session.Tokens[session.ChunkIndices[i]].Value
		if strings.HasPrefix(original, "\n") && !strings.HasPrefix(translated, "\n") {
			translated = "\n" + translated
		}
		if strings.HasSuffix(original, "\n") && !strings.HasSuffix(translated, "\n") {
			translated = translated + "\n"
		}

		session.Translations[i] = translated
	}

	return nil
}

func (s *MCPServer) saveBulkToFile() string {
	session := s.bulkSession

	// Replace text tokens with translations
	for i, idx := range session.ChunkIndices {
		if session.Translations[i] != "" {
			session.Tokens[idx].Value = session.Translations[i]
		}
	}

	// Rebuild document
	var builder strings.Builder
	for _, t := range session.Tokens {
		builder.WriteString(t.Value)
	}

	result := dropEmptyPTags(builder.String())

	// Save to file
	err := os.WriteFile(session.OutputPath, []byte(result), 0644)
	if err != nil {
		return fmt.Sprintf("ERROR guardando archivo: %v", err)
	}

	outputPath := session.OutputPath
	totalChunks := session.TotalChunks

	// Clear session
	s.bulkSession = nil

	return fmt.Sprintf(`TRADUCCION BULK COMPLETADA (ARCHIVO)
====================================
Archivo guardado: %s
Bloques traducidos: %d

El archivo Divi ha sido traducido y guardado exitosamente.
Los shortcodes [et_*] se han preservado intactos.`, outputPath, totalChunks)
}

func (s *MCPServer) saveBulkToWordPress() string {
	session := s.bulkSession

	// Replace text tokens with translations
	for i, idx := range session.ChunkIndices {
		if session.Translations[i] != "" {
			session.Tokens[idx].Value = session.Translations[i]
		}
	}

	// Rebuild document
	var builder strings.Builder
	for _, t := range session.Tokens {
		builder.WriteString(t.Value)
	}

	result := dropEmptyPTags(builder.String())

	// Update WordPress
	wpDB, err := s.getWordPressDB()
	if err != nil {
		return fmt.Sprintf("ERROR conectando a WordPress: %v", err)
	}

	err = wpDB.UpdatePostContent(session.PostID, result)
	if err != nil {
		return fmt.Sprintf("ERROR actualizando post: %v", err)
	}

	postID := session.PostID
	backupPath := session.BackupPath
	totalChunks := session.TotalChunks

	// Clear session
	s.bulkSession = nil

	return fmt.Sprintf(`TRADUCCION BULK COMPLETADA (WORDPRESS)
======================================
Post ID actualizado: %d
Backup original: %s
Bloques traducidos: %d

El post de WordPress ha sido actualizado exitosamente.
Los shortcodes [et_*] se han preservado intactos.

IMPORTANTE: Backup del contenido original en:
%s`, postID, backupPath, totalChunks, backupPath)
}

func (s *MCPServer) handleGetStatus(req JSONRPCRequest) {
	// Check bulk session first
	if s.bulkSession != nil {
		var source string
		if s.bulkSession.SourceType == "wordpress" {
			source = fmt.Sprintf("WordPress Post ID: %d", s.bulkSession.PostID)
		} else {
			source = fmt.Sprintf("Archivo: %s -> %s", s.bulkSession.InputPath, s.bulkSession.OutputPath)
		}

		response := fmt.Sprintf(`ESTADO DE TRADUCCION BULK (OPTIMIZADO)
======================================
Origen: %s
Idioma: %s
Total bloques: %d
Partes: %d
Parte actual: %d de %d`,
			source,
			s.bulkSession.TargetLang,
			s.bulkSession.TotalChunks,
			s.bulkSession.Parts,
			s.bulkSession.CurrentPart+1,
			s.bulkSession.Parts)

		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: response,
				}},
			},
		})
		return
	}

	// Check legacy session
	if s.session == nil {
		s.writeResponse(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: "No hay ninguna traduccion en progreso.",
				}},
			},
		})
		return
	}

	var source string
	if s.session.SourceType == "wordpress" {
		source = fmt.Sprintf("WordPress Post ID: %d", s.session.PostID)
	} else {
		source = fmt.Sprintf("Archivo: %s -> %s", s.session.InputPath, s.session.OutputPath)
	}

	response := fmt.Sprintf(`ESTADO DE LA TRADUCCION (LEGACY)
================================
Origen: %s
Idioma: %s
Progreso: %d/%d chunks (%d%%)`,
		source,
		s.session.TargetLang,
		s.session.CurrentChunk,
		s.session.TotalChunks,
		(s.session.CurrentChunk*100)/s.session.TotalChunks)

	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: response,
			}},
		},
	})
}

func (s *MCPServer) Run() {
	scanner := bufio.NewScanner(s.stdin)
	// Increase buffer size for large inputs
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
			continue
		}

		s.log("Received method: %s", req.Method)

		switch req.Method {
		case "initialize":
			s.handleInitialize(req)
		case "tools/list":
			s.handleListTools(req)
		case "tools/call":
			s.handleCallTool(req)
		case "notifications/initialized":
			// Client notification, no response needed
		default:
			if req.ID != nil {
				s.writeResponse(JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &RPCError{
						Code:    -32601,
						Message: fmt.Sprintf("Method not found: %s", req.Method),
					},
				})
			}
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
