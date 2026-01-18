package main

import (
	"strings"
)

// Token represents either a shortcode or text block.
type Token struct {
	Kind  string // "shortcode" or "text"
	Value string
}

// tokenize splits the input into shortcode tokens and text tokens.
// Captures both opening [et_pb_*] and closing [/et_pb_*] shortcodes.
func tokenize(input string) []Token {
	var tokens []Token
	i := 0

	for i < len(input) {
		// Find next shortcode - either [et_ or [/et_
		openIdx := strings.Index(input[i:], "[et_")
		closeIdx := strings.Index(input[i:], "[/et_")

		// Determine which comes first
		var idx int = -1
		if openIdx >= 0 && closeIdx >= 0 {
			if openIdx < closeIdx {
				idx = openIdx
			} else {
				idx = closeIdx
			}
		} else if openIdx >= 0 {
			idx = openIdx
		} else if closeIdx >= 0 {
			idx = closeIdx
		}

		if idx == -1 {
			// No more shortcodes, add remaining as text
			if i < len(input) {
				tokens = append(tokens, Token{Kind: "text", Value: input[i:]})
			}
			break
		}

		idx += i

		// Add text before shortcode
		if idx > i {
			tokens = append(tokens, Token{Kind: "text", Value: input[i:idx]})
		}

		// Find closing bracket
		end := strings.IndexByte(input[idx:], ']')
		if end == -1 {
			// Malformed, treat rest as text
			tokens = append(tokens, Token{Kind: "text", Value: input[idx:]})
			break
		}
		end += idx

		tokens = append(tokens, Token{Kind: "shortcode", Value: input[idx : end+1]})
		i = end + 1
	}

	return tokens
}

// chunkTokens groups tokens ensuring the cumulative text length per chunk stays under limit.
func chunkTokens(tokens []Token, limit int) [][]Token {
	var chunks [][]Token
	var current []Token
	var textLen int
	for _, t := range tokens {
		add := 0
		if t.Kind == "text" {
			add = len(t.Value)
		}
		if textLen+add > limit && len(current) > 0 {
			chunks = append(chunks, current)
			current = nil
			textLen = 0
		}
		current = append(current, t)
		textLen += add
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

// rebuild joins tokens back into a single string.
func rebuild(tokens []Token) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteString(t.Value)
	}
	return b.String()
}

// dropEmptyPTags removes trivial <p></p> and <p>&nbsp;</p> occurrences.
func dropEmptyPTags(s string) string {
	repl := []string{
		"<p></p>", "",
		"<p>\u00a0</p>", "",
		"<p>&nbsp;</p>", "",
	}
	r := strings.NewReplacer(repl...)
	return r.Replace(s)
}
