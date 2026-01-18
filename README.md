# MCP Divi Translator v4.0

Servidor MCP para Claude Desktop que traduce páginas Divi (shortcodes `[et_*]`) sin romper la estructura.

## Modos de operación

| Modo | Descripción | Llamadas MCP |
|------|-------------|--------------|
| **BULK (Recomendado)** | Extrae todo el texto, traduce de una vez, reensambla | 2 |
| Legacy | Traduce chunk por chunk | N+1 (puede fallar en documentos grandes) |

## Fuentes de datos

- **Archivo local** - Lee/escribe archivos `.txt`
- **WordPress MySQL** - Lee/escribe directamente en la base de datos

## Requisitos

- Go 1.21+
- Claude Desktop con soporte MCP
- (Opcional) MySQL para modo WordPress

## Instalación

```bash
cd C:\MCPs\clone\__PluginsWordpress\scp-divi-translation
go mod tidy
go build -o divi-translator.exe .
```

## Configuración en Claude Desktop

### Modo básico (solo archivos)

```json
{
  "mcpServers": {
    "divi-translator": {
      "command": "C:\\MCPs\\clone\\__PluginsWordpress\\scp-divi-translation\\divi-translator.exe",
      "args": []
    }
  }
}
```

### Modo completo (archivos + WordPress)

```json
{
  "mcpServers": {
    "divi-translator": {
      "command": "C:\\MCPs\\clone\\__PluginsWordpress\\scp-divi-translation\\divi-translator.exe",
      "args": [],
      "env": {
        "WP_MYSQL_HOST": "tu-servidor-mysql.com",
        "WP_MYSQL_PORT": "3306",
        "WP_MYSQL_USER": "usuario",
        "WP_MYSQL_PASSWORD": "contraseña",
        "WP_MYSQL_DATABASE": "nombre_bd",
        "WP_TABLE_PREFIX": "wp_",
        "WP_BACKUP_DIR": "C:\\backups\\divi"
      }
    }
  }
}
```

## Tools disponibles

### Modo BULK (Recomendado)

| Tool | Descripción |
|------|-------------|
| `extract_divi_text` | Extrae TODO el texto de un archivo en un documento con marcadores |
| `extract_wordpress_text` | Extrae TODO el texto de un post WordPress |
| `submit_bulk_translation` | Recibe la traducción completa, reensambla y guarda |

### Modo Legacy

| Tool | Descripción |
|------|-------------|
| `start_divi_translation` | Inicia traducción chunk-by-chunk desde archivo |
| `start_wordpress_translation` | Inicia traducción chunk-by-chunk desde WordPress |
| `submit_translation` | Envía un chunk traducido |

### Utilidades

| Tool | Descripción |
|------|-------------|
| `get_translation_status` | Muestra el progreso actual |

## Uso en Claude Desktop

### Modo BULK (Recomendado)

```
Traduce la página Divi usando MODO BULK:
- inputPath: C:/ruta/pagina.txt
- outputPath: C:/ruta/pagina.es.txt
- targetLang: es

1. Usa extract_divi_text para extraer el texto
2. Traduce TODO el texto manteniendo los marcadores {{CHUNK_XXX}}
3. Usa submit_bulk_translation con el texto traducido
```

### Flujo BULK

```
Claude Desktop                         Servidor MCP
      │                                     │
      ├─── extract_divi_text ──────────────►│ Lee archivo
      │                                     │ Tokeniza
      │◄─── Texto con marcadores ───────────┤ Genera marcadores
      │                                     │
      │     (Claude traduce TODO            │
      │      sin llamar herramientas)       │
      │                                     │
      ├─── submit_bulk_translation ────────►│ Parsea marcadores
      │                                     │ Reensambla
      │                                     │ Guarda archivo
      │◄─── "COMPLETADA" ───────────────────┤
      │                                     │
    TOTAL: 2 llamadas MCP
```

## Variables de entorno WordPress

| Variable | Descripción | Default |
|----------|-------------|---------|
| `WP_MYSQL_HOST` | Host del servidor MySQL | localhost |
| `WP_MYSQL_PORT` | Puerto MySQL | 3306 |
| `WP_MYSQL_USER` | Usuario MySQL | (requerido) |
| `WP_MYSQL_PASSWORD` | Contraseña MySQL | |
| `WP_MYSQL_DATABASE` | Nombre de la BD | (requerido) |
| `WP_TABLE_PREFIX` | Prefijo de tablas WP | wp_ |
| `WP_BACKUP_DIR` | Directorio para backups | . |

## Reglas de traducción

- **NO TOCAR**: shortcodes `[et_*]`, atributos `class`, `style`, `href`, `src`, `id`, `data-*`
- **TRADUCIR**: texto visible, atributos `title` y `alt`
- **CONSERVAR**: marcadores `{{CHUNK_XXX}}`, estructura HTML, saltos de línea
- **ELIMINAR**: etiquetas vacías (`<p></p>`)

## Formato de marcadores

El sistema genera texto con marcadores que deben conservarse durante la traducción:

```
{{CHUNK_001}}
<h2>Título original</h2>
<p>Texto a traducir.</p>
{{/CHUNK_001}}

{{CHUNK_002}}
<p>Más texto aquí.</p>
{{/CHUNK_002}}
```

## Particionado automático

Si el documento es muy grande (>30KB de texto), el sistema lo divide automáticamente en 2-3 partes para evitar límites de contexto.
