package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const maxCharsPerPart = 30000 // ~30KB per part, safe for Claude context
const sessionTTL = 30 * time.Minute

// generateExtractionID creates a unique ID for an extraction session
func generateExtractionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// cleanupExpiredSessions removes extraction sessions older than sessionTTL
func cleanupExpiredSessions() {
	extractionsMutex.Lock()
	defer extractionsMutex.Unlock()
	now := time.Now()
	for id, session := range activeExtractions {
		if now.Sub(session.CreatedAt) > sessionTTL {
			delete(activeExtractions, id)
		}
	}
}

// initSession initializes a legacy chunk-by-chunk translation session
func (s *MCPServer) initSession(content, targetLang, sourceType, inputPath, outputPath string, postID int64, backupPath string) {
	tokens := tokenize(content)

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

// initBulkSessionWithID creates a new bulk session with a unique ID and stores it globally
func (s *MCPServer) initBulkSessionWithID(content, targetLang, sourceType, inputPath, outputPath string, postID int64, backupPath string) *BulkTranslationSession {
	// Cleanup expired sessions before creating a new one
	cleanupExpiredSessions()

	tokens := tokenize(content)

	var chunkIndices []int
	for i, t := range tokens {
		if t.Kind == "text" && strings.TrimSpace(t.Value) != "" {
			chunkIndices = append(chunkIndices, i)
		}
	}

	if len(chunkIndices) == 0 {
		return nil
	}

	totalLen := 0
	for _, idx := range chunkIndices {
		totalLen += len(tokens[idx].Value)
	}

	parts := 1
	if totalLen > maxCharsPerPart*2 {
		parts = 3
	} else if totalLen > maxCharsPerPart {
		parts = 2
	}

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

	extractionID := generateExtractionID()

	session := &BulkTranslationSession{
		ExtractionID: extractionID,
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
		CreatedAt:    time.Now(),
	}

	extractionsMutex.Lock()
	activeExtractions[extractionID] = session
	extractionsMutex.Unlock()

	return session
}

// rebuildTranslatedDocument replaces text tokens with translations and rebuilds the document
func rebuildTranslatedDocument(tokens []Token, chunkIndices []int, translations []string) string {
	for i, idx := range chunkIndices {
		if translations[i] != "" {
			tokens[idx].Value = translations[i]
		}
	}

	var builder strings.Builder
	for _, t := range tokens {
		builder.WriteString(t.Value)
	}

	return dropEmptyPTags(builder.String())
}

// parseBulkTranslationForSession parses translated text for a specific session
func parseBulkTranslationForSession(session *BulkTranslationSession, text string) error {
	partRange := session.PartRanges[session.CurrentPart]

	// Parse WordPress metadata markers (only on first part for WordPress source)
	if session.SourceType == "wordpress" && session.CurrentPart == 0 {
		session.TranslatedTitle = parseMetadataMarker(text, "POST_TITLE", session.OriginalTitle)
		session.TranslatedSlug = parseMetadataMarker(text, "POST_SLUG", session.OriginalSlug)
		session.TranslatedExcerpt = parseMetadataMarker(text, "POST_EXCERPT", session.OriginalExcerpt)
	}

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

// parseMetadataMarker extracts a metadata value from marker tags, returning fallback if not found
func parseMetadataMarker(text, markerName, fallback string) string {
	openTag := "{{" + markerName + "}}"
	closeTag := "{{/" + markerName + "}}"

	start := strings.Index(text, openTag)
	if start == -1 {
		return fallback
	}
	end := strings.Index(text, closeTag)
	if end == -1 {
		return fallback
	}

	value := strings.TrimSpace(text[start+len(openTag) : end])
	if value == "" {
		return fallback
	}
	return value
}

// truncateForDisplay truncates a string for display purposes
func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// maskString masks a string showing only the first 4 chars
func maskString(s, defaultVal string) string {
	if s == "" {
		s = defaultVal
	}
	if s == "" {
		return "[no configurado]"
	}
	if len(s) <= 4 {
		return s + "***"
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}
