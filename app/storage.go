package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Максимальный размер файла в байтах (10MB)
const maxFileSize = 10 * 1024 * 1024

// Разрешенные типы изображений
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// ImageInfo хранит информацию об изображении
type ImageInfo struct {
	Filename string
	Path     string
	Size     int64
	UserID   string
}

// generateUniqueFilename генерирует уникальное имя файла
func generateUniqueFilename(originalFilename string, extension string) string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	randomString := hex.EncodeToString(bytes)
	
	// Получаем расширение оригинального файла
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if extension != "" {
		ext = "." + extension
	} else if ext == "" {
		ext = ".jpg" // расширение по умолчанию
	}
	
	return randomString + ext
}

// validateImageType проверяет тип изображения
func validateImageType(file multipart.File) (string, bool) {
	// Создаем буфер для чтения заголовка файла
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return "", false
	}
	
	// Восстанавливаем указатель файла в начало
	file.Seek(0, 0)
	
	// Определяем MIME тип
	contentType := http.DetectContentType(buffer)
	
	// Проверяем, является ли тип изображением
	isAllowed := allowedImageTypes[contentType]
	
	// Возвращаем соответствующее расширение
	var extension string
	switch contentType {
	case "image/jpeg":
		extension = "jpg"
	case "image/png":
		extension = "png"
	case "image/gif":
		extension = "gif"
	case "image/webp":
		extension = "webp"
	default:
		extension = ""
	}
	
	return extension, isAllowed
}

// saveImage сохраняет загруженное изображение
func saveImage(file multipart.File, header *multipart.FileHeader, userID string) (*ImageInfo, error) {
	// Проверяем размер файла
	if header.Size > maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes", header.Size)
	}
	
	// Проверяем тип изображения
	extension, valid := validateImageType(file)
	if !valid {
		return nil, fmt.Errorf("invalid image type")
	}
	
	// Генерируем уникальное имя файла
	filename := generateUniqueFilename(header.Filename, extension)
	
	// Путь для сохранения
	path := filepath.Join("/data", filename)
	
	// Создаем файл для записи
	dst, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	
	// Копируем содержимое файла
	_, err = io.Copy(dst, file)
	if err != nil {
		return nil, err
	}
	
	// Получаем информацию о файле
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	
	// Возвращаем информацию об изображении
	imageInfo := &ImageInfo{
		Filename: filename,
		Path:     path,
		Size:     stat.Size(),
		UserID:   userID,
	}
	
	return imageInfo, nil
}

// getUserImages возвращает список изображений пользователя
func getUserImages(userID string) ([]ImageInfo, error) {
	// Реализация будет добавлена в следующей фазе
	return nil, nil
}

// deleteImage удаляет изображение
func deleteImage(filename, userID string) error {
	// Реализация будет добавлена в следующей фазе
	return nil
}

// ensureDataDir создает директорию /data если она не существует
func ensureDataDir() error {
	return os.MkdirAll("/data", 0755)
}