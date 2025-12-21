package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Logger - простая структура для логирования
type Logger struct {
	debug bool
}

func NewLogger(debug bool) *Logger {
	return &Logger{debug: debug}
}

func (l *Logger) Debug(msg string) {
	if l.debug {
		log.Printf("[DEBUG] %s", msg)
	}
}

func (l *Logger) Info(msg string) {
	log.Printf("[INFO] %s", msg)
}

func (l *Logger) Error(msg string) {
	log.Printf("[ERROR] %s", msg)
}

// Global logger instance
var logger = NewLogger(true) // Включаем логирование для диагностики

// ErrorResponse отправляет JSON ответ с ошибкой
func ErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"error": "%s"}`, strings.ReplaceAll(message, `"`, `\\"`))
}

// SuccessResponse отправляет JSON ответ с успешным результатом
func SuccessResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "data": %v}`, data)
}

// ValidatePath проверяет безопасность пути
func ValidatePath(path string) bool {
	// Проверяем на попытки выйти за пределы директории
	cleanPath := filepath.Clean(path)
	return !strings.Contains(cleanPath, "..") && !strings.HasPrefix(cleanPath, "/")
}

// EnsureDir создает директорию если она не существует
func EnsureDir(path string) error {
	logger.Debug("Создание директории: " + path)
	err := os.MkdirAll(path, DefaultFilePerm)
	if err != nil {
		logger.Error("Ошибка создания директории " + path + ": " + err.Error())
	} else {
		logger.Debug("Директория успешно создана: " + path)
	}
	return err
}

// GetFileExtension возвращает расширение файла в нижнем регистре
func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	return strings.ToLower(ext)
}

// IsImageFile проверяет является ли файл изображением
func IsImageFile(filename string) bool {
	ext := GetFileExtension(filename)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}

// RandomID генерирует случайный ID
func RandomID() string {
	bytes := make([]byte, 2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback на timestamp
		return fmt.Sprintf("%04x", time.Now().UnixNano()%65536)
	}
	return hex.EncodeToString(bytes)
}