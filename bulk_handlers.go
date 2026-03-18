package main

import (
	"fmt"
	"os"
)

func (s *MCPServer) handleExtractDiviText(req JSONRPCRequest, params CallToolParams) {
	inputPath, _ := params.Arguments["inputPath"].(string)
	outputPath, _ := params.Arguments["outputPath"].(string)
	targetLang, _ := params.Arguments["targetLang"].(string)

	if inputPath == "" || outputPath == "" || targetLang == "" {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("ERROR: inputPath, outputPath y targetLang son obligatorios")})
		return
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult(fmt.Sprintf("ERROR leyendo archivo: %v", err))})
		return
	}

	session := s.initBulkSessionWithID(string(data), targetLang, "file", inputPath, outputPath, 0, "")

	if session == nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("El archivo no contiene texto para traducir (solo shortcodes).")})
		return
	}

	s.log("Sesion bulk archivo iniciada: ID=%s, %d chunks, %d partes", session.ExtractionID, session.TotalChunks, session.Parts)

	response := s.generateBulkExtractResponseWithID(session)
	s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(response)})
}

func (s *MCPServer) handleExtractWordPressText(req JSONRPCRequest, params CallToolParams) {
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

	post, err := wpDB.GetPost(postID)
	if err != nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult(fmt.Sprintf("ERROR leyendo post: %v", err))})
		return
	}

	backupPath, err := wpDB.SaveFullBackup(post, targetLang)
	if err != nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult(fmt.Sprintf("ERROR creando backup: %v", err))})
		return
	}

	session := s.initBulkSessionWithID(post.PostContent, targetLang, "wordpress", "", "", postID, backupPath)

	if session == nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("El post no contiene texto Divi para traducir.")})
		return
	}

	session.OriginalTitle = post.PostTitle
	session.OriginalSlug = post.PostName
	session.OriginalExcerpt = post.PostExcerpt

	s.log("Sesion bulk WordPress iniciada: ID=%s, Post %d, %d chunks, %d partes", session.ExtractionID, postID, session.TotalChunks, session.Parts)

	response := s.generateBulkExtractResponseWithID(session)
	s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(response)})
}

func (s *MCPServer) handleSubmitBulkTranslation(req JSONRPCRequest, params CallToolParams) {
	extractionId, _ := params.Arguments["extractionId"].(string)
	translatedText, _ := params.Arguments["translatedText"].(string)

	if extractionId == "" {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("ERROR: extractionId es obligatorio")})
		return
	}

	if translatedText == "" {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult("ERROR: translatedText es obligatorio")})
		return
	}

	extractionsMutex.RLock()
	session, exists := activeExtractions[extractionId]
	extractionsMutex.RUnlock()

	if !exists {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult(fmt.Sprintf("ERROR: extractionId '%s' no encontrado. Usa extract_divi_text o extract_wordpress_text primero.", extractionId))})
		return
	}

	err := parseBulkTranslationForSession(session, translatedText)
	if err != nil {
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: errorResult(fmt.Sprintf("ERROR parseando traduccion: %v", err))})
		return
	}

	session.CurrentPart++

	// Check if there are more parts
	if session.CurrentPart < session.Parts {
		response := s.generateBulkExtractResponseWithID(session)
		s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(fmt.Sprintf("PARTE %d RECIBIDA\n\n%s", session.CurrentPart, response))})
		return
	}

	// All parts received, save the result
	var result string
	if session.SourceType == "wordpress" {
		result = s.saveBulkToWordPress(session)
	} else {
		result = s.saveBulkToFile(session)
	}

	// Remove from active extractions
	extractionsMutex.Lock()
	delete(activeExtractions, extractionId)
	extractionsMutex.Unlock()

	s.writeResponse(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: textResult(result)})
}

func (s *MCPServer) saveBulkToFile(session *BulkTranslationSession) string {
	result := rebuildTranslatedDocument(session.Tokens, session.ChunkIndices, session.Translations)

	err := os.WriteFile(session.OutputPath, []byte(result), 0644)
	if err != nil {
		return fmt.Sprintf("ERROR guardando archivo: %v", err)
	}

	return fmt.Sprintf(`TRADUCCION BULK COMPLETADA (ARCHIVO)
====================================
extractionId: %s
Archivo guardado: %s
Bloques traducidos: %d

El archivo Divi ha sido traducido y guardado exitosamente.
Los shortcodes [et_*] se han preservado intactos.`, session.ExtractionID, session.OutputPath, session.TotalChunks)
}

func (s *MCPServer) saveBulkToWordPress(session *BulkTranslationSession) string {
	translatedContent := rebuildTranslatedDocument(session.Tokens, session.ChunkIndices, session.Translations)

	wpDB, err := s.getWordPressDB()
	if err != nil {
		return fmt.Sprintf("ERROR conectando a WordPress: %v", err)
	}

	err = wpDB.UpdatePostFull(
		session.PostID,
		session.TranslatedTitle,
		session.TranslatedSlug,
		session.TranslatedExcerpt,
		translatedContent,
	)
	if err != nil {
		return fmt.Sprintf("ERROR actualizando post: %v", err)
	}

	return fmt.Sprintf(`TRADUCCION BULK COMPLETADA (WORDPRESS)
======================================
extractionId: %s
Post ID actualizado: %d
Backup original: %s
Bloques traducidos: %d

CAMPOS ACTUALIZADOS:
- Titulo: %s
- Slug: %s
- Excerpt: %s
- Contenido: %d bloques traducidos

El post de WordPress ha sido actualizado exitosamente.
Los shortcodes [et_*] se han preservado intactos.

IMPORTANTE: Backup del contenido original en:
%s`, session.ExtractionID, session.PostID, session.BackupPath, session.TotalChunks,
		session.TranslatedTitle, session.TranslatedSlug,
		truncateForDisplay(session.TranslatedExcerpt, 50), session.TotalChunks, session.BackupPath)
}
