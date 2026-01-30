#!/bin/bash
# Script de instalación para MCP Divi Translator en macOS
# Uso: ./install-macos.sh

set -e

echo "========================================="
echo "MCP Divi Translator - Instalación macOS"
echo "========================================="
echo ""

# Colores para output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Detectar directorio del proyecto
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
echo "📁 Directorio del proyecto: $PROJECT_DIR"
echo ""

# Verificar Go
echo "🔍 Verificando instalación de Go..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go no está instalado${NC}"
    echo "Instala Go desde: https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo -e "${GREEN}✓ Go instalado: $GO_VERSION${NC}"
echo ""

# Verificar arquitectura
ARCH=$(uname -m)
if [ "$ARCH" != "arm64" ]; then
    echo -e "${YELLOW}⚠️  Advertencia: Este script está optimizado para arm64/Apple Silicon${NC}"
    echo "   Tu arquitectura: $ARCH"
    echo ""
fi

# Compilar
echo "🔨 Compilando servidor MCP..."
cd "$PROJECT_DIR"
go mod download
go build -o divi-translator .
chmod +x divi-translator

if [ ! -f "divi-translator" ]; then
    echo -e "${RED}❌ Error al compilar${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Compilación exitosa${NC}"
echo ""

# Configurar Claude Desktop
CLAUDE_CONFIG="$HOME/Library/Application Support/Claude/claude_desktop_config.json"
echo "⚙️  Configurando Claude Desktop..."

# Crear directorio si no existe
mkdir -p "$HOME/Library/Application Support/Claude"

# Verificar si el archivo existe
if [ -f "$CLAUDE_CONFIG" ]; then
    echo -e "${YELLOW}⚠️  El archivo de configuración ya existe${NC}"
    echo "   Ruta: $CLAUDE_CONFIG"
    echo ""
    echo "¿Quieres actualizar la configuración? (s/n)"
    read -r response
    if [[ ! "$response" =~ ^[Ss]$ ]]; then
        echo "Instalación cancelada. Configura manualmente según INSTALL_MACOS.md"
        exit 0
    fi
    # Hacer backup
    cp "$CLAUDE_CONFIG" "$CLAUDE_CONFIG.backup.$(date +%Y%m%d_%H%M%S)"
    echo -e "${GREEN}✓ Backup creado${NC}"
fi

# Preguntar si usa WordPress
echo ""
echo "¿Vas a usar el modo WordPress (requiere MySQL)? (s/n)"
read -r use_wordpress

if [[ "$use_wordpress" =~ ^[Ss]$ ]]; then
    echo ""
    echo "Introduce los datos de conexión MySQL:"
    echo -n "Host (default: localhost): "
    read -r mysql_host
    mysql_host=${mysql_host:-localhost}

    echo -n "Puerto (default: 3306): "
    read -r mysql_port
    mysql_port=${mysql_port:-3306}

    echo -n "Usuario: "
    read -r mysql_user

    echo -n "Contraseña: "
    read -rs mysql_password
    echo ""

    echo -n "Base de datos: "
    read -r mysql_database

    echo -n "Prefijo de tablas (default: wp_): "
    read -r table_prefix
    table_prefix=${table_prefix:-wp_}

    echo -n "Directorio de backups (default: $HOME/backups/divi): "
    read -r backup_dir
    backup_dir=${backup_dir:-$HOME/backups/divi}

    # Crear directorio de backups
    mkdir -p "$backup_dir"

    # Crear configuración completa
    cat > "$CLAUDE_CONFIG" <<EOF
{
  "mcpServers": {
    "divi-translator": {
      "command": "$PROJECT_DIR/divi-translator",
      "args": [],
      "env": {
        "WP_MYSQL_HOST": "$mysql_host",
        "WP_MYSQL_PORT": "$mysql_port",
        "WP_MYSQL_USER": "$mysql_user",
        "WP_MYSQL_PASSWORD": "$mysql_password",
        "WP_MYSQL_DATABASE": "$mysql_database",
        "WP_TABLE_PREFIX": "$table_prefix",
        "WP_BACKUP_DIR": "$backup_dir"
      }
    }
  }
}
EOF
else
    # Crear configuración básica
    cat > "$CLAUDE_CONFIG" <<EOF
{
  "mcpServers": {
    "divi-translator": {
      "command": "$PROJECT_DIR/divi-translator",
      "args": []
    }
  }
}
EOF
fi

echo ""
echo -e "${GREEN}✓ Configuración de Claude Desktop actualizada${NC}"
echo ""

# Verificar si Claude Desktop está corriendo
if pgrep -x "Claude" > /dev/null; then
    echo -e "${YELLOW}⚠️  Claude Desktop está en ejecución${NC}"
    echo "   Necesitas reiniciar Claude Desktop para que los cambios surtan efecto"
    echo ""
    echo "   Pasos:"
    echo "   1. Cierra Claude Desktop (⌘Q)"
    echo "   2. Abre Claude Desktop de nuevo"
fi

echo ""
echo "========================================="
echo -e "${GREEN}✅ Instalación completada${NC}"
echo "========================================="
echo ""
echo "📋 Resumen:"
echo "   Binario: $PROJECT_DIR/divi-translator"
echo "   Config:  $CLAUDE_CONFIG"
if [[ "$use_wordpress" =~ ^[Ss]$ ]]; then
    echo "   Backups: $backup_dir"
fi
echo ""
echo "📖 Documentación:"
echo "   - INSTALL_MACOS.md: Guía completa de instalación"
echo "   - CLAUDE.md: Instrucciones de uso"
echo "   - README.md: Descripción del proyecto"
echo ""
echo "🧪 Para verificar:"
echo "   1. Reinicia Claude Desktop"
echo "   2. Pregunta: '¿Qué herramientas MCP están disponibles?'"
echo "   3. Deberías ver: extract_divi_text, submit_bulk_translation, etc."
echo ""
echo "🎉 ¡Listo para traducir páginas Divi!"
echo ""
