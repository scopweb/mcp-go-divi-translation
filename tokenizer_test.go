package main

import (
	"testing"
)

func TestTokenize_BasicShortcodes(t *testing.T) {
	input := `[et_pb_section]Hello world[/et_pb_section]`
	tokens := tokenize(input)

	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != "shortcode" || tokens[0].Value != "[et_pb_section]" {
		t.Errorf("token 0: got %q %q", tokens[0].Kind, tokens[0].Value)
	}
	if tokens[1].Kind != "text" || tokens[1].Value != "Hello world" {
		t.Errorf("token 1: got %q %q", tokens[1].Kind, tokens[1].Value)
	}
	if tokens[2].Kind != "shortcode" || tokens[2].Value != "[/et_pb_section]" {
		t.Errorf("token 2: got %q %q", tokens[2].Kind, tokens[2].Value)
	}
}

func TestTokenize_NoShortcodes(t *testing.T) {
	input := `Just plain text`
	tokens := tokenize(input)

	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	if tokens[0].Kind != "text" || tokens[0].Value != "Just plain text" {
		t.Errorf("got %q %q", tokens[0].Kind, tokens[0].Value)
	}
}

func TestTokenize_EmptyInput(t *testing.T) {
	tokens := tokenize("")
	if len(tokens) != 0 {
		t.Fatalf("expected 0 tokens, got %d", len(tokens))
	}
}

func TestTokenize_NestedAttributes(t *testing.T) {
	input := `[et_pb_text _builder_version="4.0"]<p>Hello</p>[/et_pb_text]`
	tokens := tokenize(input)

	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Value != `[et_pb_text _builder_version="4.0"]` {
		t.Errorf("token 0: got %q", tokens[0].Value)
	}
	if tokens[1].Value != "<p>Hello</p>" {
		t.Errorf("token 1: got %q", tokens[1].Value)
	}
}

func TestTokenize_Roundtrip(t *testing.T) {
	input := `[et_pb_section][et_pb_row][et_pb_text]<h2>Title</h2><p>Body</p>[/et_pb_text][/et_pb_row][/et_pb_section]`
	tokens := tokenize(input)
	result := rebuild(tokens)

	if result != input {
		t.Errorf("roundtrip failed:\ngot:  %q\nwant: %q", result, input)
	}
}

func TestDropEmptyPTags(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"<p></p>", ""},
		{"<p>&nbsp;</p>", ""},
		{"<p>\u00a0</p>", ""},
		{"<p>Keep me</p>", "<p>Keep me</p>"},
		{"<p></p>Hello<p></p>", "Hello"},
	}

	for _, tt := range tests {
		got := dropEmptyPTags(tt.input)
		if got != tt.want {
			t.Errorf("dropEmptyPTags(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestChunkTokens(t *testing.T) {
	tokens := []Token{
		{Kind: "shortcode", Value: "[et_pb_text]"},
		{Kind: "text", Value: "AAAA"},
		{Kind: "shortcode", Value: "[/et_pb_text]"},
		{Kind: "shortcode", Value: "[et_pb_text]"},
		{Kind: "text", Value: "BBBB"},
		{Kind: "shortcode", Value: "[/et_pb_text]"},
	}

	// Limit of 5 should split into 2 chunks (4 chars each text)
	chunks := chunkTokens(tokens, 5)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
}
