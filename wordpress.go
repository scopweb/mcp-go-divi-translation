package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// WordPressDB handles WordPress database operations
type WordPressDB struct {
	db          *sql.DB
	tablePrefix string
	backupDir   string
}

// WordPressPost represents a WordPress post
type WordPressPost struct {
	ID          int64
	PostTitle   string
	PostName    string // slug
	PostExcerpt string
	PostContent string
	PostStatus  string
	PostType    string
}

// NewWordPressDB creates a new WordPress database connection
func NewWordPressDB() (*WordPressDB, error) {
	host := os.Getenv("WP_MYSQL_HOST")
	port := os.Getenv("WP_MYSQL_PORT")
	user := os.Getenv("WP_MYSQL_USER")
	password := os.Getenv("WP_MYSQL_PASSWORD")
	database := os.Getenv("WP_MYSQL_DATABASE")
	tablePrefix := os.Getenv("WP_TABLE_PREFIX")
	backupDir := os.Getenv("WP_BACKUP_DIR")

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "3306"
	}
	if tablePrefix == "" {
		tablePrefix = "wp_"
	}
	if backupDir == "" {
		backupDir = "."
	}

	if user == "" || database == "" {
		return nil, fmt.Errorf("WP_MYSQL_USER y WP_MYSQL_DATABASE son obligatorios")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true",
		user, password, host, port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error conectando a MySQL: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error verificando conexión MySQL: %v", err)
	}

	return &WordPressDB{
		db:          db,
		tablePrefix: tablePrefix,
		backupDir:   backupDir,
	}, nil
}

// Close closes the database connection
func (wp *WordPressDB) Close() {
	if wp.db != nil {
		wp.db.Close()
	}
}

// GetPost retrieves a WordPress post by ID
func (wp *WordPressDB) GetPost(postID int64) (*WordPressPost, error) {
	query := fmt.Sprintf(`
		SELECT ID, post_title, post_name, post_excerpt, post_content, post_status, post_type
		FROM %sposts
		WHERE ID = ?`,
		wp.tablePrefix)

	post := &WordPressPost{}
	err := wp.db.QueryRow(query, postID).Scan(
		&post.ID,
		&post.PostTitle,
		&post.PostName,
		&post.PostExcerpt,
		&post.PostContent,
		&post.PostStatus,
		&post.PostType,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post ID %d no encontrado", postID)
		}
		return nil, fmt.Errorf("error leyendo post: %v", err)
	}

	return post, nil
}

// UpdatePostContent updates the post_content of a WordPress post
func (wp *WordPressDB) UpdatePostContent(postID int64, newContent string) error {
	query := fmt.Sprintf(`
		UPDATE %sposts
		SET post_content = ?, post_modified = NOW(), post_modified_gmt = UTC_TIMESTAMP()
		WHERE ID = ?`,
		wp.tablePrefix)

	result, err := wp.db.Exec(query, newContent, postID)
	if err != nil {
		return fmt.Errorf("error actualizando post: %v", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("post ID %d no encontrado para actualizar", postID)
	}

	return nil
}

// UpdatePostFull updates post_content, post_title, post_name (slug), and post_excerpt
func (wp *WordPressDB) UpdatePostFull(postID int64, title, slug, excerpt, content string) error {
	query := fmt.Sprintf(`
		UPDATE %sposts
		SET post_title = ?, post_name = ?, post_excerpt = ?, post_content = ?,
		    post_modified = NOW(), post_modified_gmt = UTC_TIMESTAMP()
		WHERE ID = ?`,
		wp.tablePrefix)

	result, err := wp.db.Exec(query, title, slug, excerpt, content, postID)
	if err != nil {
		return fmt.Errorf("error actualizando post: %v", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("post ID %d no encontrado para actualizar", postID)
	}

	return nil
}

// SaveBackup saves the original content to a backup file
func (wp *WordPressDB) SaveBackup(postID int64, content string, lang string) (string, error) {
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(wp.backupDir, 0755); err != nil {
		return "", fmt.Errorf("error creando directorio de backup: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("post_%d_backup_%s_%s.txt", postID, lang, timestamp)
	backupPath := filepath.Join(wp.backupDir, filename)

	if err := os.WriteFile(backupPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("error guardando backup: %v", err)
	}

	return backupPath, nil
}

// SaveFullBackup saves all translatable fields to a backup file
func (wp *WordPressDB) SaveFullBackup(post *WordPressPost, lang string) (string, error) {
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(wp.backupDir, 0755); err != nil {
		return "", fmt.Errorf("error creando directorio de backup: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("post_%d_full_backup_%s_%s.txt", post.ID, lang, timestamp)
	backupPath := filepath.Join(wp.backupDir, filename)

	// Create backup content with all fields
	backupContent := fmt.Sprintf(`=== WORDPRESS POST BACKUP ===
Post ID: %d
Date: %s
Target Language: %s

=== POST_TITLE ===
%s

=== POST_NAME (SLUG) ===
%s

=== POST_EXCERPT ===
%s

=== POST_CONTENT ===
%s
`, post.ID, timestamp, lang, post.PostTitle, post.PostName, post.PostExcerpt, post.PostContent)

	if err := os.WriteFile(backupPath, []byte(backupContent), 0644); err != nil {
		return "", fmt.Errorf("error guardando backup: %v", err)
	}

	return backupPath, nil
}

// TranslateAndUpdatePost handles the complete flow: read, backup, translate, update
// This is designed to work with the existing translation session
func (wp *WordPressDB) ReadPostForTranslation(postID int64, targetLang string) (*WordPressPost, string, error) {
	// Get the post
	post, err := wp.GetPost(postID)
	if err != nil {
		return nil, "", err
	}

	// Save backup
	backupPath, err := wp.SaveBackup(postID, post.PostContent, targetLang)
	if err != nil {
		return nil, "", err
	}

	return post, backupPath, nil
}

// GetTablePrefix returns the configured table prefix
func (wp *WordPressDB) GetTablePrefix() string {
	return wp.tablePrefix
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(s string) string {
	// Replace invalid characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := s
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Limit length
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}
