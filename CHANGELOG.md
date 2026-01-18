# Changelog

Todos los cambios notables de este proyecto se documentan en este archivo.

## [4.0.0] - 2025-01-18

### Agregado

- **Modo BULK (Optimizado)**: Nuevo flujo de traduccion que reduce drasticamente las llamadas MCP
  - `extract_divi_text`: Extrae TODO el texto de un archivo Divi en un solo documento con marcadores `{{CHUNK_XXX}}`
  - `extract_wordpress_text`: Igual pero desde WordPress MySQL
  - `submit_bulk_translation`: Recibe el texto traducido completo, parsea los marcadores y reensambla el documento

- **Particionado automatico**: Si el texto supera 30KB, se divide automaticamente en 2-3 partes

- **Marcadores de chunk**: Sistema de marcadores `{{CHUNK_001}}...{{/CHUNK_001}}` para identificar bloques de texto

### Cambiado

- Flujo de traduccion optimizado: de N+1 llamadas MCP a solo 2 llamadas
- Actualizado README.md con documentacion del modo BULK
- Actualizado CLAUDE.md con instrucciones para Claude Desktop

### Por que este cambio?

El modo legacy (chunk-by-chunk) requeria una llamada MCP por cada bloque de texto. En documentos grandes (60+ chunks), Claude Desktop perdia el contexto y fallaba antes de completar la traduccion.

El nuevo modo BULK:
1. Extrae todo el texto en una sola llamada
2. Claude traduce sin llamar herramientas (minimo consumo de tokens)
3. Reensambla en una sola llamada

**Resultado**: Traducciones que antes fallaban ahora completan exitosamente.

---

## [3.0.0] - 2025-01-17

### Agregado

- Soporte para WordPress MySQL directo
  - `start_wordpress_translation`: Lee posts directamente de la BD
  - Backup automatico del contenido original
  - Actualizacion automatica de la BD al finalizar

- Variables de entorno para configuracion MySQL:
  - `WP_MYSQL_HOST`, `WP_MYSQL_PORT`, `WP_MYSQL_USER`
  - `WP_MYSQL_PASSWORD`, `WP_MYSQL_DATABASE`
  - `WP_TABLE_PREFIX`, `WP_BACKUP_DIR`

### Cambiado

- Refactorizado el sistema de sesiones para soportar multiples fuentes

---

## [2.0.0] - 2025-01-16

### Agregado

- Sistema de tokenizacion mejorado para shortcodes Divi
- Soporte para shortcodes de cierre `[/et_*]`
- Limpieza automatica de etiquetas HTML vacias

### Cambiado

- Mejor manejo de caracteres especiales en atributos

---

## [1.0.0] - 2025-01-15

### Agregado

- Implementacion inicial del servidor MCP
- `start_divi_translation`: Inicia traduccion desde archivo
- `submit_translation`: Envia traduccion de un chunk
- `get_translation_status`: Muestra progreso
- Tokenizador basico para shortcodes `[et_*]`
