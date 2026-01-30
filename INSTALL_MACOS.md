# Instalación en macOS (arm64)

## Requisitos

- macOS (arm64/Apple Silicon)
- Go 1.25.6 (ya instalado ✓)
- Claude Desktop para Mac

## Pasos de instalación

### 1. Transferir el proyecto al Mac

Transfiere toda la carpeta del proyecto a tu Mac. Por ejemplo:

```bash
# En tu Mac, crea un directorio para MCPs
mkdir -p ~/MCPs
cd ~/MCPs

# Clona o copia el proyecto aquí
# La ruta final será algo como: ~/MCPs/scp-divi-translation
```

### 2. Compilar el servidor MCP para arm64

```bash
cd ~/MCPs/scp-divi-translation

# Descargar dependencias
go mod download

# Compilar para macOS arm64
go build -o divi-translator .

# Verificar que se compiló correctamente
./divi-translator --version || echo "Binary compilado exitosamente"

# Dar permisos de ejecución
chmod +x divi-translator
```

El binario resultante `divi-translator` estará optimizado para Apple Silicon (arm64).

### 3. Configurar Claude Desktop

Edita el archivo de configuración de Claude Desktop en Mac:

```bash
# El archivo está en:
~/Library/Application Support/Claude/claude_desktop_config.json
```

Abre el archivo con tu editor favorito:

```bash
# Opción 1: nano
nano ~/Library/Application\ Support/Claude/claude_desktop_config.json

# Opción 2: VS Code
code ~/Library/Application\ Support/Claude/claude_desktop_config.json

# Opción 3: TextEdit
open -a TextEdit ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

### 4. Configuración básica (solo archivos)

```json
{
  "mcpServers": {
    "divi-translator": {
      "command": "/Users/TU_USUARIO/MCPs/scp-divi-translation/divi-translator",
      "args": []
    }
  }
}
```

**IMPORTANTE**: Reemplaza `TU_USUARIO` con tu nombre de usuario real de macOS.

Para obtener la ruta completa exacta:

```bash
cd ~/MCPs/scp-divi-translation
pwd
# Copia el resultado y agrégale /divi-translator al final
```

### 5. Configuración completa (archivos + WordPress)

```json
{
  "mcpServers": {
    "divi-translator": {
      "command": "/Users/TU_USUARIO/MCPs/scp-divi-translation/divi-translator",
      "args": [],
      "env": {
        "WP_MYSQL_HOST": "tu-servidor-mysql.com",
        "WP_MYSQL_PORT": "3306",
        "WP_MYSQL_USER": "usuario",
        "WP_MYSQL_PASSWORD": "contraseña",
        "WP_MYSQL_DATABASE": "nombre_bd",
        "WP_TABLE_PREFIX": "wp_",
        "WP_BACKUP_DIR": "/Users/TU_USUARIO/backups/divi"
      }
    }
  }
}
```

### 6. Crear directorio de backups (opcional, si usas WordPress)

```bash
mkdir -p ~/backups/divi
chmod 755 ~/backups/divi
```

### 7. Reiniciar Claude Desktop

Después de guardar la configuración:

1. Cierra completamente Claude Desktop (⌘Q)
2. Abre Claude Desktop de nuevo
3. El servidor MCP debería cargarse automáticamente

### 8. Verificar instalación

En Claude Desktop, pregunta:

```
¿Qué herramientas MCP están disponibles?
```

Deberías ver las herramientas del divi-translator:
- `extract_divi_text`
- `extract_wordpress_text`
- `submit_bulk_translation`
- `get_translation_status`
- etc.

## Solución de problemas

### Error: "Permission denied"

```bash
chmod +x ~/MCPs/scp-divi-translation/divi-translator
```

### Error: "command not found" en Claude Desktop

Verifica que la ruta en `claude_desktop_config.json` sea absoluta y correcta:

```bash
# Obtener ruta absoluta
cd ~/MCPs/scp-divi-translation
echo "$(pwd)/divi-translator"
```

Copia esa ruta exacta en tu configuración.

### Ver logs de Claude Desktop

```bash
# Los logs están en:
tail -f ~/Library/Logs/Claude/mcp*.log
```

### MySQL no conecta

Verifica que tu servidor MySQL sea accesible desde tu Mac:

```bash
# Test de conexión
mysql -h TU_HOST -P 3306 -u TU_USUARIO -p
```

## Rutas en macOS vs Windows

| Windows | macOS |
|---------|-------|
| `C:\MCPs\...` | `~/MCPs/...` o `/Users/tu_usuario/MCPs/...` |
| `C:\backups\divi` | `~/backups/divi` o `/Users/tu_usuario/backups/divi` |
| `.exe` | Sin extensión |
| `\` separador | `/` separador |

## Ejemplo de uso completo

Una vez configurado, en Claude Desktop:

```
Traduce la página Divi a español:
- inputPath: ~/Documents/pagina.txt
- outputPath: ~/Documents/pagina.es.txt
- targetLang: es

Usa extract_divi_text, traduce manteniendo los marcadores {{CHUNK_XXX}},
y usa submit_bulk_translation.
```

## Notas adicionales

- El binario compilado (`divi-translator`) es específico para arm64, no funcionará en Mac Intel
- Si necesitas compilar para Mac Intel, usa: `GOARCH=amd64 go build -o divi-translator-amd64 .`
- El proyecto es compatible con Go 1.21+ y tu versión 1.25.6 es perfecta
- No necesitas instalar dependencias adicionales, todo está en el módulo Go

## Próximos pasos

1. Transfiere el proyecto
2. Compila con `go build`
3. Configura Claude Desktop
4. ¡Listo para traducir!
