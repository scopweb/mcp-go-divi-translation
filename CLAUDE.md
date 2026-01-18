# Instrucciones para Claude Desktop - MCP Divi Translator

Este servidor MCP traduce paginas Divi automaticamente preservando los shortcodes `[et_*]`.

## MODO RECOMENDADO: Bulk Translation

Usa SIEMPRE el modo BULK. El modo legacy (chunk-by-chunk) puede fallar en documentos grandes.

### Flujo de trabajo

```
1. extract_divi_text / extract_wordpress_text  →  Recibir extractionId + texto con marcadores
2. Traducir el texto (SIN llamar herramientas)
3. submit_bulk_translation(extractionId, texto) →  Enviar traduccion completa
```

### IMPORTANTE: extractionId

Cada extraccion devuelve un `extractionId` unico (ej: `41167e5c31ab74c1`).
DEBES usar este ID al llamar `submit_bulk_translation`.

### Prompt ejemplo para archivo

```
Traduce la pagina Divi a [IDIOMA]:
- inputPath: [RUTA_ENTRADA]
- outputPath: [RUTA_SALIDA]
- targetLang: [CODIGO_IDIOMA]

Usa extract_divi_text, traduce manteniendo los marcadores {{CHUNK_XXX}}, y usa submit_bulk_translation.
```

### Prompt ejemplo para WordPress

```
Traduce el post de WordPress a [IDIOMA]:
- postId: [ID_POST]
- targetLang: [CODIGO_IDIOMA]

Usa extract_wordpress_text, traduce manteniendo los marcadores {{CHUNK_XXX}}, {{POST_TITLE}}, {{POST_SLUG}}, {{POST_EXCERPT}}, y usa submit_bulk_translation.
```

## Metadatos de WordPress (Title, Slug, Excerpt)

Para posts de WordPress, `extract_wordpress_text` incluye los metadatos del post para traducir:

```
METADATOS DEL POST (traducir tambien):
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

**IMPORTANTE para metadatos:**
- `POST_TITLE`: Traducir el titulo del post
- `POST_SLUG`: Traducir/adaptar el slug (URL amigable). Usar guiones, sin espacios, sin caracteres especiales
- `POST_EXCERPT`: Traducir el extracto/resumen del post

**Ejemplo de traduccion de metadatos:**
```
{{POST_TITLE}}
Mi Titulo Traducido
{{/POST_TITLE}}

{{POST_SLUG}}
mi-titulo-traducido
{{/POST_SLUG}}

{{POST_EXCERPT}}
Este es el extracto traducido del post.
{{/POST_EXCERPT}}
```

## Reglas de traduccion CRITICAS

### TRADUCIR
- Texto visible entre etiquetas HTML
- Atributos `alt` de imagenes
- Atributos `title`

### NO TRADUCIR (preservar exactamente)
- Shortcodes Divi: `[et_pb_section]`, `[et_pb_row]`, `[et_pb_text]`, etc.
- Shortcodes de cierre: `[/et_pb_section]`, `[/et_pb_text]`, etc.
- Atributos de shortcodes: `_builder_version`, `global_colors_info`, etc.
- Atributos HTML: `class`, `style`, `href`, `src`, `id`, `data-*`, `width`, `height`
- URLs completas
- Marcadores de contenido: `{{CHUNK_001}}`, `{{/CHUNK_001}}`
- Marcadores de metadatos: `{{POST_TITLE}}`, `{{POST_SLUG}}`, `{{POST_EXCERPT}}` y sus cierres

### CONSERVAR
- Estructura HTML exacta
- Saltos de linea
- Espaciado
- Entidades HTML (`&nbsp;`, `&amp;`, etc.)

### ELIMINAR
- Etiquetas vacias resultantes: `<p></p>`, `<span></span>`

## Formato de marcadores

El servidor genera texto con este formato:

```
{{CHUNK_001}}
<h2>Titulo en idioma original</h2>
<p>Parrafo a traducir.</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<p>Otro bloque de texto.</p>
{{/CHUNK_002}}
```

Tu traduccion debe mantener los marcadores:

```
{{CHUNK_001}}
<h2>Titulo traducido</h2>
<p>Parrafo traducido.</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<p>Otro bloque traducido.</p>
{{/CHUNK_002}}
```

## Tools disponibles

| Tool | Uso | Descripcion |
|------|-----|-------------|
| `extract_divi_text` | BULK | Extrae texto de archivo, devuelve extractionId + texto |
| `extract_wordpress_text` | BULK | Extrae texto de WordPress, devuelve extractionId + texto |
| `submit_bulk_translation` | BULK | Recibe extractionId + traduccion, guarda resultado |
| `get_translation_status` | Info | Muestra progreso actual |
| `start_divi_translation` | Legacy | NO USAR - puede fallar |
| `start_wordpress_translation` | Legacy | NO USAR - puede fallar |
| `submit_translation` | Legacy | NO USAR |

### Parametros de submit_bulk_translation

```json
{
  "extractionId": "41167e5c31ab74c1",
  "translatedText": "{{CHUNK_001}}\n<p>Texto traducido...</p>\n{{/CHUNK_001}}"
}
```

## Codigos de idioma comunes

| Codigo | Idioma |
|--------|--------|
| `es` | Espanol |
| `en` | Ingles |
| `fr` | Frances |
| `de` | Aleman |
| `it` | Italiano |
| `pt` | Portugues |
| `nl` | Holandes |
| `ca` | Catalan |

## Ejemplo completo de traduccion

**Entrada (extract_divi_text devuelve):**
```
{{CHUNK_001}}
<p class="intro">Welcome to our store</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<img src="image.jpg" alt="Beautiful landscape" />
<p>Contact us today!</p>
{{/CHUNK_002}}
```

**Salida correcta (para submit_bulk_translation):**
```
{{CHUNK_001}}
<p class="intro">Bienvenido a nuestra tienda</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<img src="image.jpg" alt="Hermoso paisaje" />
<p>Contactanos hoy!</p>
{{/CHUNK_002}}
```

Nota: `class="intro"` y `src="image.jpg"` NO se traducen. `alt="Beautiful landscape"` SI se traduce.

## Ejemplo completo WordPress (con metadatos)

**Entrada (extract_wordpress_text devuelve):**
```
METADATOS DEL POST (traducir tambien):
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

CONTENIDO A TRADUCIR:
=====================

{{CHUNK_001}}
<p class="intro">Welcome to our store</p>
{{/CHUNK_001}}
```

**Salida correcta (para submit_bulk_translation):**
```
{{POST_TITLE}}
Bienvenido a Nuestra Tienda
{{/POST_TITLE}}

{{POST_SLUG}}
bienvenido-a-nuestra-tienda
{{/POST_SLUG}}

{{POST_EXCERPT}}
Descubre nuestros increibles productos y servicios.
{{/POST_EXCERPT}}

{{CHUNK_001}}
<p class="intro">Bienvenido a nuestra tienda</p>
{{/CHUNK_001}}
```

## Manejo de partes multiples

Si el documento es muy grande, el servidor puede dividirlo en partes:

```
PARTE 1 de 3
============
{{CHUNK_001}}...{{/CHUNK_001}}
{{CHUNK_002}}...{{/CHUNK_002}}
...
```

Traduce cada parte y envia con `submit_bulk_translation`. El servidor pedira la siguiente parte hasta completar.

## Errores comunes a evitar

1. **NO traducir marcadores**: `{{CHUNK_001}}` debe quedar exactamente igual
2. **NO traducir URLs**: `https://example.com` debe quedar igual
3. **NO traducir clases CSS**: `class="wp-image-123"` debe quedar igual
4. **NO anadir texto extra**: Solo traducir, no agregar explicaciones
5. **NO omitir marcadores**: Cada `{{CHUNK_XXX}}` debe tener su `{{/CHUNK_XXX}}`
