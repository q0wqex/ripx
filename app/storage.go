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
	
	// Создаем директорию пользователя если она не существует
	err := ensureUserDir(userID)
	if err != nil {
		return nil, err
	}
	
	// Генерируем уникальное имя файла
	filename := generateUniqueFilename(header.Filename, extension)
	
	// Путь для сохранения в директории пользователя
	path := filepath.Join("/data", userID, filename)
	
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
	return getUserImagesPaginated(userID, 0, 0)
}

// getUserImagesPaginated возвращает список изображений пользователя с пагинацией
// Если pageSize <= 0, возвращает все изображения
func getUserImagesPaginated(userID string, page, pageSize int) ([]ImageInfo, error) {
	userDir := filepath.Join("/data", userID)
	
	// Проверяем существование директории
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		return []ImageInfo{}, nil
	}
	
	// Читаем содержимое директории
	entries, err := os.ReadDir(userDir)
	if err != nil {
		return nil, err
	}
	
	var images []ImageInfo
	for _, entry := range entries {
		// Пропускаем директории
		if entry.IsDir() {
			continue
		}
		
		// Проверяем, что файл является изображением по расширению
		filename := entry.Name()
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
			continue
		}
		
		// Получаем информацию о файле
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		// Создаем путь к файлу
		path := filepath.Join(userDir, filename)
		
		// Добавляем в список
		images = append(images, ImageInfo{
			Filename: filename,
			Path:     path,
			Size:     info.Size(),
			UserID:   userID,
		})
	}
	
	// Применяем пагинацию
	if pageSize > 0 {
		start := page * pageSize
		if start >= len(images) {
			return []ImageInfo{}, nil
		}
		
		end := start + pageSize
		if end > len(images) {
			end = len(images)
		}
		
		images = images[start:end]
	}
	
	return images, nil
}

// deleteImage удаляет изображение
func deleteImage(filename, userID string) error {
	// Реализация будет добавлена в следующей фазе
	return nil
}

// getSessionID получает или генерирует ID сессии пользователя
func getSessionID(w http.ResponseWriter, r *http.Request) string {
	// Проверяем наличие cookie
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	
	// Генерируем новый ID сессии
	bytes := make([]byte, 16)
	rand.Read(bytes)
	sessionID := hex.EncodeToString(bytes)
	
	// Устанавливаем cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   86400 * 30, // 30 дней
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	
	return sessionID
}

// ensureDataDir создает директорию /data если она не существует
func ensureDataDir() error {
	return os.MkdirAll("/data", 0755)
}

// ensureUserDir создает директорию для пользователя если она не существует
func ensureUserDir(userID string) error {
	return os.MkdirAll(filepath.Join("/data", userID), 0755)
}