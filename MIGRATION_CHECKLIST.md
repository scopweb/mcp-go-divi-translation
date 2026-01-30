# ✅ Checklist de Migración a macOS

## Pasos rápidos

### 1️⃣ Transferir archivos al Mac

Opciones para transferir:

**Opción A: Git (recomendado)**
```bash
# En tu Mac
cd ~/MCPs
git clone [URL_DEL_REPO]
```

**Opción B: Compresión y transferencia**
```bash
# En Windows (desde este directorio)
tar -czf divi-translator.tar.gz .

# Luego transferir el archivo .tar.gz a tu Mac
# En Mac:
cd ~/MCPs
tar -xzf ~/Downloads/divi-translator.tar.gz
```

**Opción C: AirDrop/USB**
- Copia toda la carpeta `scp-divi-translation` a tu Mac

### 2️⃣ Ejecutar instalación automatizada

```bash
# En tu Mac
cd ~/MCPs/scp-divi-translation
chmod +x install-macos.sh
./install-macos.sh
```

El script hará:
- ✓ Verificar Go 1.25.6
- ✓ Compilar para arm64
- ✓ Configurar Claude Desktop
- ✓ (Opcional) Configurar MySQL

### 3️⃣ Reiniciar Claude Desktop

1. Cerrar Claude Desktop: `⌘Q`
2. Abrir Claude Desktop de nuevo
3. Verificar en Claude: "¿Qué herramientas MCP están disponibles?"

## Archivos importantes para el Mac

```
scp-divi-translation/
├── divi-translator          ← Binario compilado (se genera)
├── install-macos.sh         ← Script de instalación ✨
├── INSTALL_MACOS.md         ← Guía detallada
├── CLAUDE.md                ← Instrucciones de uso
├── main.go                  ← Código fuente
├── go.mod                   ← Dependencias (ya limpiadas ✓)
└── go.sum                   ← Checksums
```

## Diferencias Windows → macOS

| Aspecto | Windows | macOS |
|---------|---------|-------|
| Binario | `divi-translator.exe` | `divi-translator` |
| Separador | `\` | `/` |
| Rutas | `C:\MCPs\...` | `~/MCPs/...` |
| Config | `%APPDATA%\Claude\...` | `~/Library/Application Support/Claude/...` |
| Ejecutar | `divi-translator.exe` | `./divi-translator` |

## Versiones compatibles

- ✅ Go 1.25.6 (tu versión) - Compatible
- ✅ arm64 (Apple Silicon) - Optimizado
- ✅ macOS reciente - Compatible

## Solución rápida de problemas

### Problema: "permission denied"
```bash
chmod +x ~/MCPs/scp-divi-translation/divi-translator
chmod +x ~/MCPs/scp-divi-translation/install-macos.sh
```

### Problema: "command not found" en Claude
```bash
# Obtener ruta absoluta correcta
cd ~/MCPs/scp-divi-translation
echo "$(pwd)/divi-translator"
# Copiar esa ruta exacta en claude_desktop_config.json
```

### Ver logs de Claude Desktop
```bash
tail -f ~/Library/Logs/Claude/mcp*.log
```

## Después de la instalación

### Prueba básica (archivos)
```
En Claude Desktop:

Traduce esta página Divi a español:
- inputPath: /ruta/a/archivo.txt
- outputPath: /ruta/a/archivo.es.txt
- targetLang: es
```

### Prueba avanzada (WordPress)
```
En Claude Desktop:

Traduce el post de WordPress #123 a español:
- postId: 123
- targetLang: es
```

## 📋 Checklist final

- [ ] Proyecto transferido a `~/MCPs/scp-divi-translation`
- [ ] Ejecutado `./install-macos.sh` exitosamente
- [ ] Claude Desktop reiniciado
- [ ] Herramientas MCP visibles en Claude
- [ ] (Opcional) MySQL configurado y accesible
- [ ] Primera traducción de prueba completada

## 🎯 ¿Todo listo?

Si marcaste todos los checkboxes, ¡estás listo para usar el MCP Divi Translator en tu Mac!

Consulta `CLAUDE.md` para ejemplos de uso completos.
