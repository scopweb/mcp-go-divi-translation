package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

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

// isValidRequestID ensures ID is a string or integer (not null)
func isValidRequestID(id interface{}) bool {
	if id == nil {
		return false
	}
	switch id.(type) {
	case string, float64, int, int64:
		return true
	default:
		return false
	}
}

// rpcError creates a JSON-RPC error response with the given code, message, and request ID
func (s *MCPServer) rpcError(code int, message string, id interface{}) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message},
	}
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

// errorResult creates a CallToolResult with an error message
func errorResult(msg string) CallToolResult {
	return CallToolResult{
		Content: []ContentItem{{Type: "text", Text: msg}},
		IsError: true,
	}
}

// textResult creates a CallToolResult with a text message
func textResult(msg string) CallToolResult {
	return CallToolResult{
		Content: []ContentItem{{Type: "text", Text: msg}},
	}
}

func (s *MCPServer) handleInitialize(req JSONRPCRequest) {
	var params InitializeParams
	if req.Params != nil {
		json.Unmarshal(req.Params, &params)
	}

	// Echo client's protocol version for universal compatibility
	// Fallback to latest if client doesn't send one
	negotiatedVersion := params.ProtocolVersion
	if negotiatedVersion == "" {
		negotiatedVersion = MCP_PROTOCOL_VERSION
	}

	result := InitializeResult{
		ProtocolVersion: negotiatedVersion,
	}
	result.Capabilities.Tools = map[string]interface{}{}
	result.ServerInfo.Name = "divi-translator"
	result.ServerInfo.Version = "4.4.0"

	s.initialized = true

	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	})
}

func (s *MCPServer) handleListTools(req JSONRPCRequest) {
	tools := []Tool{
		{
			Name:        "start_divi_translation",
			Description: "Inicia la traduccion de una pagina Divi DESDE ARCHIVO. Devuelve el primer chunk de texto a traducir.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"inputPath":  map[string]interface{}{"type": "string", "description": "Ruta absoluta del fichero Divi a traducir"},
					"outputPath": map[string]interface{}{"type": "string", "description": "Ruta donde guardar el fichero traducido"},
					"targetLang": map[string]interface{}{"type": "string", "description": "Codigo de idioma destino (es, en, fr, de, it, pt, etc.)"},
				},
				"required": []string{"inputPath", "outputPath", "targetLang"},
			},
		},
		{
			Name:        "start_wordpress_translation",
			Description: "Inicia la traduccion de un post de WordPress DESDE BASE DE DATOS. Lee el post, crea backup y devuelve el primer chunk. Al finalizar actualiza automaticamente la BD.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"postId":     map[string]interface{}{"type": "integer", "description": "ID del post de WordPress a traducir"},
					"targetLang": map[string]interface{}{"type": "string", "description": "Codigo de idioma destino (es, en, fr, de, it, pt, etc.)"},
				},
				"required": []string{"postId", "targetLang"},
			},
		},
		{
			Name:        "submit_translation",
			Description: "Envia la traduccion del chunk actual. Devuelve el siguiente chunk, o confirma que se guardo (archivo o BD) si era el ultimo.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"translatedText": map[string]interface{}{"type": "string", "description": "El texto traducido del chunk actual (solo el texto, sin marcadores)"},
				},
				"required": []string{"translatedText"},
			},
		},
		{
			Name:        "get_translation_status",
			Description: "Obtiene el estado actual de la traduccion en progreso.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		// Bulk translation (optimized)
		{
			Name:        "extract_divi_text",
			Description: "OPTIMIZADO: Extrae texto de archivo Divi. Devuelve extractionId y texto con marcadores {{CHUNK_XXX}}. Traduce el texto y usa submit_bulk_translation con el extractionId.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"inputPath":  map[string]interface{}{"type": "string", "description": "Ruta absoluta del archivo Divi a procesar"},
					"outputPath": map[string]interface{}{"type": "string", "description": "Ruta donde guardar el archivo traducido final"},
					"targetLang": map[string]interface{}{"type": "string", "description": "Codigo de idioma destino (es, en, fr, de, etc.)"},
				},
				"required": []string{"inputPath", "outputPath", "targetLang"},
			},
		},
		{
			Name:        "extract_wordpress_text",
			Description: "OPTIMIZADO: Extrae texto de post WordPress. Devuelve extractionId y texto con marcadores {{CHUNK_XXX}}. Traduce el texto y usa submit_bulk_translation con el extractionId.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"postId":     map[string]interface{}{"type": "integer", "description": "ID del post de WordPress"},
					"targetLang": map[string]interface{}{"type": "string", "description": "Codigo de idioma destino (es, en, fr, de, etc.)"},
				},
				"required": []string{"postId", "targetLang"},
			},
		},
		{
			Name:        "server_info",
			Description: "Devuelve informacion del servidor: version, estado de conexion MySQL, configuracion activa y tools disponibles.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "submit_bulk_translation",
			Description: "Recibe extractionId y texto traducido (con marcadores {{CHUNK_XXX}}), reensambla y guarda el documento.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"extractionId":  map[string]interface{}{"type": "string", "description": "ID de extraccion devuelto por extract_divi_text o extract_wordpress_text"},
					"translatedText": map[string]interface{}{"type": "string", "description": "El texto traducido completo, manteniendo los marcadores {{CHUNK_XXX}}...{{/CHUNK_XXX}}"},
				},
				"required": []string{"extractionId", "translatedText"},
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
		s.writeResponse(s.rpcError(ErrInvalidParams, fmt.Sprintf("Invalid params: %v", err), req.ID))
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
	case "extract_divi_text":
		s.handleExtractDiviText(req, params)
	case "extract_wordpress_text":
		s.handleExtractWordPressText(req, params)
	case "submit_bulk_translation":
		s.handleSubmitBulkTranslation(req, params)
	case "server_info":
		s.handleServerInfo(req)
	default:
		s.writeResponse(s.rpcError(ErrMethodNotFound, fmt.Sprintf("Unknown tool: %s", params.Name), req.ID))
	}
}

func (s *MCPServer) handleStartTranslation(req JSONRPCRequest, params CallToolParams) {
	inputPath, _ := params.Arguments["inputPath"].(string)
	outputPath, _ := params.Arguments["outputPath"].(string)
	targetLang, _ := params.Arguments["targetLang"].(string)

	if inputPath == "" || outputPath == "" || targetLang == "" {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("ERROR: inputPath, outputPath y targetLang son obligatorios")})
		return
	}

	if !isSafePath(inputPath) || !isSafePath(outputPath) {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("ERROR: ruta de archivo no segura")})
		return
	}

	// #gosec G304 // path validated by isSafePath check above
	data, err := os.ReadFile(inputPath)
	if err != nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult(fmt.Sprintf("ERROR leyendo fichero: %v", err))})
		return
	}

	s.initSession(string(data), targetLang, "file", inputPath, outputPath, 0, "")

	if s.session == nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("El archivo no contiene texto para traducir (solo shortcodes).")})
		return
	}

	s.log("Sesion archivo iniciada: %d tokens, %d chunks de texto", len(s.session.Tokens), s.session.TotalChunks)

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

	s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(response)})
}

func (s *MCPServer) handleStartWordPressTranslation(req JSONRPCRequest, params CallToolParams) {
	postIDFloat, _ := params.Arguments["postId"].(float64)
	postID := int64(postIDFloat)
	targetLang, _ := params.Arguments["targetLang"].(string)

	if postID == 0 || targetLang == "" {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("ERROR: postId y targetLang son obligatorios")})
		return
	}

	wpDB, err := s.getWordPressDB()
	if err != nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult(fmt.Sprintf("ERROR conectando a WordPress: %v", err))})
		return
	}

	post, backupPath, err := wpDB.ReadPostForTranslation(postID, targetLang)
	if err != nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult(fmt.Sprintf("ERROR leyendo post: %v", err))})
		return
	}

	s.initSession(post.PostContent, targetLang, "wordpress", "", "", postID, backupPath)

	if s.session == nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("El post no contiene texto Divi para traducir (solo shortcodes).")})
		return
	}

	s.log("Sesion WordPress iniciada: Post %d, %d tokens, %d chunks de texto", postID, len(s.session.Tokens), s.session.TotalChunks)

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

	s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(response)})
}

func (s *MCPServer) handleSubmitTranslation(req JSONRPCRequest, params CallToolParams) {
	if s.session == nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("ERROR: No hay ninguna traduccion en progreso. Usa start_divi_translation o start_wordpress_translation primero.")})
		return
	}

	translatedText, _ := params.Arguments["translatedText"].(string)
	if translatedText == "" {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("ERROR: translatedText es obligatorio")})
		return
	}

	s.session.Translations[s.session.CurrentChunk] = translatedText
	s.session.CurrentChunk++

	s.log("Chunk %d/%d traducido", s.session.CurrentChunk, s.session.TotalChunks)

	if s.session.CurrentChunk >= s.session.TotalChunks {
		var result string
		if s.session.SourceType == "wordpress" {
			result = s.saveToWordPress()
		} else {
			result = s.saveTranslatedFile()
		}
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(result)})
		return
	}

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

	s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(response)})
}

func (s *MCPServer) saveTranslatedFile() string {
	result := rebuildTranslatedDocument(s.session.Tokens, s.session.ChunkIndices, s.session.Translations)

	err := os.WriteFile(s.session.OutputPath, []byte(result), 0600)
	if err != nil {
		return fmt.Sprintf("ERROR guardando archivo: %v", err)
	}

	outputPath := s.session.OutputPath
	totalChunks := s.session.TotalChunks
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
	result := rebuildTranslatedDocument(s.session.Tokens, s.session.ChunkIndices, s.session.Translations)

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

func (s *MCPServer) handleGetStatus(req JSONRPCRequest) {
	// Check active bulk extractions
	extractionsMutex.RLock()
	activeSessions := len(activeExtractions)
	extractionsMutex.RUnlock()

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
Parte actual: %d de %d
Sesiones activas: %d`,
			source, s.bulkSession.TargetLang, s.bulkSession.TotalChunks,
			s.bulkSession.Parts, s.bulkSession.CurrentPart+1, s.bulkSession.Parts, activeSessions)

		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(response)})
		return
	}

	if s.session == nil {
		msg := "No hay ninguna traduccion en progreso."
		if activeSessions > 0 {
			msg = fmt.Sprintf("No hay traduccion legacy en progreso. Sesiones bulk activas: %d", activeSessions)
		}
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(msg)})
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
		source, s.session.TargetLang,
		s.session.CurrentChunk, s.session.TotalChunks,
		(s.session.CurrentChunk*100)/s.session.TotalChunks)

	s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(response)})
}

func (s *MCPServer) handleServerInfo(req JSONRPCRequest) {
	mysqlStatus := "OK"
	mysqlDB := os.Getenv("WP_MYSQL_DATABASE")
	if mysqlDB == "" {
		mysqlDB = "(no configurado)"
	}
	wpDB, err := s.getWordPressDB()
	if err != nil {
		mysqlStatus = fmt.Sprintf("ERROR: %v", err)
	} else {
		if pingErr := wpDB.db.Ping(); pingErr != nil {
			mysqlStatus = fmt.Sprintf("ERROR ping: %v", pingErr)
		}
	}

	host := maskString(os.Getenv("WP_MYSQL_HOST"), "localhost")
	port := os.Getenv("WP_MYSQL_PORT")
	if port == "" {
		port = "3306"
	}
	tablePrefix := os.Getenv("WP_TABLE_PREFIX")
	if tablePrefix == "" {
		tablePrefix = "wp_"
	}
	backupDir := os.Getenv("WP_BACKUP_DIR")
	if backupDir == "" {
		backupDir = "."
	}
	mysqlDB = maskString(mysqlDB, "")

	extractionsMutex.RLock()
	activeSessions := len(activeExtractions)
	extractionsMutex.RUnlock()

	info := fmt.Sprintf(`=== DIVI TRANSLATOR SERVER INFO ===
Version:          4.4.0
Protocol:         %s

--- MySQL ---
Status:           %s
Host:             %s:%s
Database:         %s
Table Prefix:     %s

--- Rutas ---
Backup Dir:       %s

--- Sesiones activas ---
Bulk extractions: %d

--- Tools disponibles ---
  Bulk (recomendado):
    extract_divi_text
    extract_wordpress_text
    submit_bulk_translation
  Utilidad:
    get_translation_status
    server_info
  Legacy (deprecated):
    start_divi_translation
    start_wordpress_translation
    submit_translation`,
		MCP_PROTOCOL_VERSION, mysqlStatus, host, port, mysqlDB, tablePrefix, backupDir, activeSessions)

	s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(info)})
}

func (s *MCPServer) handlePing(req JSONRPCRequest) {
	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{},
	})
}

func (s *MCPServer) handleShutdown(req JSONRPCRequest) {
	if s.wpDB != nil {
		s.wpDB.Close()
	}

	s.writeResponse(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{},
	})

	s.shouldShutdown = true
}

func (s *MCPServer) getSourceDescriptionForSession(session *BulkTranslationSession) string {
	if session.SourceType == "wordpress" {
		return fmt.Sprintf("WordPress Post ID %d", session.PostID)
	}
	return session.InputPath
}

// generateBulkExtractResponseWithID generates the extraction response with extractionId included
func (s *MCPServer) generateBulkExtractResponseWithID(session *BulkTranslationSession) string {
	partRange := session.PartRanges[session.CurrentPart]

	var builder strings.Builder

	if session.Parts == 1 {
		builder.WriteString(fmt.Sprintf(`EXTRACCION COMPLETADA
=====================
extractionId: %s
Origen: %s
Idioma destino: %s
Total de bloques: %d

INSTRUCCIONES:
1. Traduce TODO el texto a %s
2. CONSERVA los marcadores {{CHUNK_XXX}} y {{/CHUNK_XXX}} exactamente igual
3. NO traduzcas atributos HTML (class, style, href, src, id, data-*)
4. SI traduce atributos "title" y "alt"
5. Conserva la estructura HTML y saltos de linea
6. Usa "submit_bulk_translation" con extractionId="%s" y el texto traducido
`, session.ExtractionID, s.getSourceDescriptionForSession(session), session.TargetLang, session.TotalChunks,
			session.TargetLang, session.ExtractionID))
	} else {
		builder.WriteString(fmt.Sprintf(`EXTRACCION COMPLETADA - PARTE %d de %d
======================================
extractionId: %s
Origen: %s
Idioma destino: %s
Bloques en esta parte: %d-%d de %d total

INSTRUCCIONES:
1. Traduce TODO el texto a %s
2. CONSERVA los marcadores {{CHUNK_XXX}} y {{/CHUNK_XXX}} exactamente igual
3. NO traduzcas atributos HTML (class, style, href, src, id, data-*)
4. SI traduce atributos "title" y "alt"
5. Conserva la estructura HTML y saltos de linea
6. Usa "submit_bulk_translation" con extractionId="%s" y el texto traducido
`, session.CurrentPart+1, session.Parts, session.ExtractionID, s.getSourceDescriptionForSession(session), session.TargetLang,
			partRange[0]+1, partRange[1], session.TotalChunks, session.TargetLang, session.ExtractionID))
	}

	// Add WordPress metadata section for first part only
	if session.SourceType == "wordpress" && session.CurrentPart == 0 {
		builder.WriteString(fmt.Sprintf(`
METADATOS DEL POST (traducir tambien):
======================================
{{POST_TITLE}}
%s
{{/POST_TITLE}}

{{POST_SLUG}}
%s
{{/POST_SLUG}}

{{POST_EXCERPT}}
%s
{{/POST_EXCERPT}}

`, session.OriginalTitle, session.OriginalSlug, session.OriginalExcerpt))
	}

	// Content section header
	if session.Parts == 1 {
		builder.WriteString("CONTENIDO A TRADUCIR:\n=====================\n")
	} else {
		builder.WriteString(fmt.Sprintf("CONTENIDO A TRADUCIR (PARTE %d):\n================================\n", session.CurrentPart+1))
	}

	// Generate text blocks with markers
	for i := partRange[0]; i < partRange[1]; i++ {
		chunkIdx := session.ChunkIndices[i]
		text := session.Tokens[chunkIdx].Value
		builder.WriteString(fmt.Sprintf("\n{{CHUNK_%03d}}\n%s\n{{/CHUNK_%03d}}\n", i+1, text, i+1))
	}

	return builder.String()
}
