package main

import (
	"testing"
	"time"
)

func TestGenerateExtractionID(t *testing.T) {
	id1 := generateExtractionID()
	id2 := generateExtractionID()

	if len(id1) != 16 {
		t.Errorf("expected 16 char hex ID, got %q (len %d)", id1, len(id1))
	}
	if id1 == id2 {
		t.Error("expected unique IDs, got duplicates")
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	// Clear state
	extractionsMutex.Lock()
	activeExtractions = make(map[string]*BulkTranslationSession)
	extractionsMutex.Unlock()

	// Add an expired session
	extractionsMutex.Lock()
	activeExtractions["expired"] = &BulkTranslationSession{
		ExtractionID: "expired",
		CreatedAt:    time.Now().Add(-2 * time.Hour),
	}
	activeExtractions["fresh"] = &BulkTranslationSession{
		ExtractionID: "fresh",
		CreatedAt:    time.Now(),
	}
	extractionsMutex.Unlock()

	cleanupExpiredSessions()

	extractionsMutex.RLock()
	defer extractionsMutex.RUnlock()

	if _, exists := activeExtractions["expired"]; exists {
		t.Error("expired session should have been cleaned up")
	}
	if _, exists := activeExtractions["fresh"]; !exists {
		t.Error("fresh session should still exist")
	}
}

func TestParseMetadataMarker(t *testing.T) {
	text := `{{POST_TITLE}}
Mi Titulo Traducido
{{/POST_TITLE}}

{{POST_SLUG}}
mi-titulo-traducido
{{/POST_SLUG}}`

	title := parseMetadataMarker(text, "POST_TITLE", "fallback")
	if title != "Mi Titulo Traducido" {
		t.Errorf("expected 'Mi Titulo Traducido', got %q", title)
	}

	slug := parseMetadataMarker(text, "POST_SLUG", "fallback")
	if slug != "mi-titulo-traducido" {
		t.Errorf("expected 'mi-titulo-traducido', got %q", slug)
	}

	// Missing marker returns fallback
	missing := parseMetadataMarker(text, "POST_EXCERPT", "my-fallback")
	if missing != "my-fallback" {
		t.Errorf("expected 'my-fallback', got %q", missing)
	}
}

func TestTruncateForDisplay(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"hello world", 8, "hello..."},
		{"ab", 2, "ab"},
		{"abc", 3, "abc"},
	}

	for _, tt := range tests {
		got := truncateForDisplay(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateForDisplay(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		input, def, want string
	}{
		{"", "", "[no configurado]"},
		{"", "default", "defa***"},
		{"ab", "", "ab***"},
		{"abcdef", "", "abcd**"},
	}

	for _, tt := range tests {
		got := maskString(tt.input, tt.def)
		if got != tt.want {
			t.Errorf("maskString(%q, %q) = %q, want %q", tt.input, tt.def, got, tt.want)
		}
	}
}

func TestParseBulkTranslationForSession(t *testing.T) {
	session := &BulkTranslationSession{
		Tokens: []Token{
			{Kind: "shortcode", Value: "[et_pb_text]"},
			{Kind: "text", Value: "Hello world"},
			{Kind: "shortcode", Value: "[/et_pb_text]"},
			{Kind: "shortcode", Value: "[et_pb_text]"},
			{Kind: "text", Value: "Goodbye world"},
			{Kind: "shortcode", Value: "[/et_pb_text]"},
		},
		ChunkIndices: []int{1, 4},
		TotalChunks:  2,
		Parts:        1,
		CurrentPart:  0,
		PartRanges:   [][2]int{{0, 2}},
		Translations: make([]string, 2),
		SourceType:   "file",
	}

	translatedText := `{{CHUNK_001}}
Hola mundo
{{/CHUNK_001}}

{{CHUNK_002}}
Adios mundo
{{/CHUNK_002}}`

	err := parseBulkTranslationForSession(session, translatedText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.Translations[0] != "Hola mundo" {
		t.Errorf("translation 0: got %q, want 'Hola mundo'", session.Translations[0])
	}
	if session.Translations[1] != "Adios mundo" {
		t.Errorf("translation 1: got %q, want 'Adios mundo'", session.Translations[1])
	}
}

func TestRebuildTranslatedDocument(t *testing.T) {
	tokens := []Token{
		{Kind: "shortcode", Value: "[et_pb_text]"},
		{Kind: "text", Value: "Hello"},
		{Kind: "shortcode", Value: "[/et_pb_text]"},
	}

	result := rebuildTranslatedDocument(tokens, []int{1}, []string{"Hola"})
	want := "[et_pb_text]Hola[/et_pb_text]"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}
