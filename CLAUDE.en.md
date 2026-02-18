# Instructions for Claude Desktop - MCP Divi Translator

This MCP server automatically translates Divi pages while preserving `[et_*]` shortcodes.

## RECOMMENDED MODE: Bulk Translation

ALWAYS use BULK mode. Legacy mode (chunk-by-chunk) can fail on large documents.

### Workflow

```
1. extract_divi_text / extract_wordpress_text  →  Receive extractionId + text with markers
2. Translate the text (WITHOUT calling tools)
3. submit_bulk_translation(extractionId, text) →  Send complete translation
```

### IMPORTANT: extractionId

Each extraction returns a unique `extractionId` (e.g.: `41167e5c31ab74c1`).
YOU MUST use this ID when calling `submit_bulk_translation`.

### Example prompt for file

```
Translate the Divi page to [LANGUAGE]:
- inputPath: [INPUT_PATH]
- outputPath: [OUTPUT_PATH]
- targetLang: [LANGUAGE_CODE]

Use extract_divi_text, translate while keeping {{CHUNK_XXX}} markers, and use submit_bulk_translation.
```

### Example prompt for WordPress

```
Translate the WordPress post to [LANGUAGE]:
- postId: [POST_ID]
- targetLang: [LANGUAGE_CODE]

Use extract_wordpress_text, translate while keeping {{CHUNK_XXX}}, {{POST_TITLE}}, {{POST_SLUG}}, {{POST_EXCERPT}} markers, and use submit_bulk_translation.
```

## WordPress Metadata (Title, Slug, Excerpt)

For WordPress posts, `extract_wordpress_text` includes post metadata for translation:

```
POST METADATA (translate too):
======================================
{{POST_TITLE}}
My Original Title
{{/POST_TITLE}}

{{POST_SLUG}}
my-original-slug
{{/POST_SLUG}}

{{POST_EXCERPT}}
This is the original excerpt of the post.
{{/POST_EXCERPT}}
```

**IMPORTANT for metadata:**
- `POST_TITLE`: Translate the post title
- `POST_SLUG`: Translate/adapt the slug (friendly URL). Use hyphens, no spaces, no special characters
- `POST_EXCERPT`: Translate the post excerpt/summary

**Example of metadata translation:**
```
{{POST_TITLE}}
My Translated Title
{{/POST_TITLE}}

{{POST_SLUG}}
my-translated-title
{{/POST_SLUG}}

{{POST_EXCERPT}}
This is the translated excerpt of the post.
{{/POST_EXCERPT}}
```

## CRITICAL Translation Rules

### TRANSLATE
- Visible text between HTML tags
- Image `alt` attributes
- `title` attributes

### DO NOT TRANSLATE (preserve exactly)
- Divi shortcodes: `[et_pb_section]`, `[et_pb_row]`, `[et_pb_text]`, etc.
- Closing shortcodes: `[/et_pb_section]`, `[/et_pb_text]`, etc.
- Shortcode attributes: `_builder_version`, `global_colors_info`, etc.
- HTML attributes: `class`, `style`, `href`, `src`, `id`, `data-*`, `width`, `height`
- Full URLs
- Content markers: `{{CHUNK_001}}`, `{{/CHUNK_001}}`
- Metadata markers: `{{POST_TITLE}}`, `{{POST_SLUG}}`, `{{POST_EXCERPT}}` and their closing tags

### PRESERVE
- Exact HTML structure
- Line breaks
- Spacing
- HTML entities (`&nbsp;`, `&amp;`, etc.)

### REMOVE
- Empty tags resulting from translation: `<p></p>`, `<span></span>`

## Marker Format

The server generates text with this format:

```
{{CHUNK_001}}
<h2>Title in original language</h2>
<p>Paragraph to translate.</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<p>Another block of text.</p>
{{/CHUNK_002}}
```

Your translation must preserve the markers:

```
{{CHUNK_001}}
<h2>Translated title</h2>
<p>Translated paragraph.</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<p>Another translated block.</p>
{{/CHUNK_002}}
```

## Available Tools

| Tool | Usage | Description |
|------|-------|-------------|
| `extract_divi_text` | BULK | Extracts text from file, returns extractionId + text |
| `extract_wordpress_text` | BULK | Extracts text from WordPress, returns extractionId + text |
| `submit_bulk_translation` | BULK | Receives extractionId + translation, saves result |
| `get_translation_status` | Info | Shows current progress |
| `server_info` | Info | Returns server version, MySQL status, and available tools |
| `start_divi_translation` | Legacy | DO NOT USE - may fail |
| `start_wordpress_translation` | Legacy | DO NOT USE - may fail |
| `submit_translation` | Legacy | DO NOT USE |

### Parameters for submit_bulk_translation

```json
{
  "extractionId": "41167e5c31ab74c1",
  "translatedText": "{{CHUNK_001}}\n<p>Translated text...</p>\n{{/CHUNK_001}}"
}
```

## Common Language Codes

| Code | Language |
|------|----------|
| `es` | Spanish |
| `en` | English |
| `fr` | French |
| `de` | German |
| `it` | Italian |
| `pt` | Portuguese |
| `nl` | Dutch |
| `ca` | Catalan |

## Complete Translation Example

**Input (extract_divi_text returns):**
```
{{CHUNK_001}}
<p class="intro">Welcome to our store</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<img src="image.jpg" alt="Beautiful landscape" />
<p>Contact us today!</p>
{{/CHUNK_002}}
```

**Correct output (for submit_bulk_translation):**
```
{{CHUNK_001}}
<p class="intro">Bienvenido a nuestra tienda</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<img src="image.jpg" alt="Hermoso paisaje" />
<p>Contactanos hoy!</p>
{{/CHUNK_002}}
```

Note: `class="intro"` and `src="image.jpg"` are NOT translated. `alt="Beautiful landscape"` IS translated.

## Complete WordPress Example (with metadata)

**Input (extract_wordpress_text returns):**
```
POST METADATA (translate too):
======================================
{{POST_TITLE}}
Welcome to Our Store
{{/POST_TITLE}}

{{POST_SLUG}}
welcome-to-our-store
{{/POST_SLUG}}

{{POST_EXCERPT}}
Discover our amazing products and services.
{{/POST_EXCERPT}}

CONTENT TO TRANSLATE:
=====================

{{CHUNK_001}}
<p class="intro">Welcome to our store</p>
{{/CHUNK_001}}
```

**Correct output (for submit_bulk_translation):**
```
{{POST_TITLE}}
Welcome to Our Store
{{/POST_TITLE}}

{{POST_SLUG}}
welcome-to-our-store
{{/POST_SLUG}}

{{POST_EXCERPT}}
Discover our amazing products and services.
{{/POST_EXCERPT}}

{{CHUNK_001}}
<p class="intro">Bienvenido a nuestra tienda</p>
{{/CHUNK_001}}
```

## Handling Multiple Parts

If the document is very large, the server may split it into parts:

```
PART 1 of 3
============
{{CHUNK_001}}...{{/CHUNK_001}}
{{CHUNK_002}}...{{/CHUNK_002}}
...
```

Translate each part and send with `submit_bulk_translation`. The server will request the next part until complete.

## Common Mistakes to Avoid

1. **DO NOT translate markers**: `{{CHUNK_001}}` must remain exactly the same
2. **DO NOT translate URLs**: `https://example.com` must remain the same
3. **DO NOT translate CSS classes**: `class="wp-image-123"` must remain the same
4. **DO NOT add extra text**: Only translate, don't add explanations
5. **DO NOT omit markers**: Each `{{CHUNK_XXX}}` must have its `{{/CHUNK_XXX}}`
