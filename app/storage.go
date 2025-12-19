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
	"time"
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
	AlbumID  string
}

// AlbumInfo хранит информацию об альбоме
type AlbumInfo struct {
	ID         string
	Name       string
	ImageCount int
}

// generateUniqueFilename генерирует уникальное имя файла
func generateUniqueFilename(originalFilename string, extension string) string {
	// Получаем расширение оригинального файла
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if extension != "" {
		ext = "." + extension
	} else if ext == "" {
		ext = ".jpg" // расширение по умолчанию
	}
	
	// Генерируем 4-символьный hex ID (2 байта)
	bytes := make([]byte, 2)
	_, err := rand.Read(bytes)
	var randomString string
	if err != nil {
		// В случае ошибки генерации случайных данных, используем timestamp
		randomString = fmt.Sprintf("%04x", time.Now().UnixNano()%65536)
	} else {
		randomString = hex.EncodeToString(bytes)
	}
	
	filename := randomString + ext
	return filename
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
	_, err = file.Seek(0, 0)
	if err != nil {
		return "", false
	}
	
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
func saveImage(file multipart.File, header *multipart.FileHeader, userID string, albumID string) (*ImageInfo, error) {
	// Проверяем размер файла
	if header.Size > maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes", header.Size)
	}
	
	// Проверяем тип изображения
	extension, valid := validateImageType(file)
	if !valid {
		return nil, fmt.Errorf("invalid image type")
	}
	
	// Создаем директорию для альбома если она не существует
	err := ensureAlbumDir(userID, albumID)
	if err != nil {
		return nil, err
	}
	
	// Генерируем уникальное имя файла
	filename := generateUniqueFilename(header.Filename, extension)
	
	// Путь для сохранения в директории альбома
	path := filepath.Join("/data", userID, albumID, filename)
	
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
		AlbumID:  albumID,
	}
	
	return imageInfo, nil
}

// getUserImages возвращает список изображений пользователя
func getUserImages(userID string) ([]ImageInfo, error) {
	return getUserImagesPaginated(userID, "", 0, 0)
}

// getUserImagesPaginated возвращает список изображений пользователя с пагинацией
// Если pageSize <= 0, возвращает все изображения
func getUserImagesPaginated(userID string, albumID string, page, pageSize int) ([]ImageInfo, error) {
	var dirPath string
	
	// Если albumID указан, используем директорию альбома, иначе директорию пользователя
	if albumID != "" {
		dirPath = filepath.Join("/data", userID, albumID)
	} else {
		dirPath = filepath.Join("/data", userID)
	}
	
	// Проверяем существование директории
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return []ImageInfo{}, nil
	}
	
	// Читаем содержимое директории
	entries, err := os.ReadDir(dirPath)
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
		path := filepath.Join(dirPath, filename)
		
		// Добавляем в список
		images = append(images, ImageInfo{
			Filename: filename,
			Path:     path,
			Size:     info.Size(),
			UserID:   userID,
			AlbumID:  albumID,
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


// getSessionID получает или генерирует ID сессии пользователя
func getSessionID(w http.ResponseWriter, r *http.Request) string {
	// Проверяем наличие cookie
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	
	// Генерируем новый ID сессии (4 символа hex = 2 байта)
	bytes := make([]byte, 2)
	_, err = rand.Read(bytes)
	var sessionID string
	if err != nil {
		// В случае ошибки генерации случайных данных, используем timestamp
		sessionID = fmt.Sprintf("%04x", time.Now().UnixNano()%65536)
	} else {
		sessionID = hex.EncodeToString(bytes)
	}
	
	fmt.Printf("[DEBUG] Generated sessionID: %s\n", sessionID)
	
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

// ensureAlbumDir создает директорию для альбома пользователя если она не существует
func ensureAlbumDir(userID, albumID string) error {
	return os.MkdirAll(filepath.Join("/data", userID, albumID), 0755)
}

// getUserAlbums возвращает список альбомов пользователя
func getUserAlbums(userID string) ([]AlbumInfo, error) {
	userDir := filepath.Join("/data", userID)
	fmt.Printf("[DEBUG] getUserAlbums: checking directory %s\n", userDir)
	
	// Проверяем существование директории
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		fmt.Printf("[DEBUG] getUserAlbums: directory does not exist\n")
		return []AlbumInfo{}, nil
	}
	
	// Читаем содержимое директории
	entries, err := os.ReadDir(userDir)
	if err != nil {
		fmt.Printf("[DEBUG] getUserAlbums: error reading directory: %v\n", err)
		return nil, err
	}
	fmt.Printf("[DEBUG] getUserAlbums: found %d entries in directory\n", len(entries))
	
	var albums []AlbumInfo
	for _, entry := range entries {
		fmt.Printf("[DEBUG] getUserAlbums: processing entry %s, isDir=%v\n", entry.Name(), entry.IsDir())
		
		// Нас интересуют только директории (альбомы)
		if !entry.IsDir() {
			continue
		}
		
		albumID := entry.Name()
		albumDir := filepath.Join(userDir, albumID)
		
		// Подсчитываем количество файлов в директории альбома
		albumEntries, err := os.ReadDir(albumDir)
		if err != nil {
			continue
		}
		
		imageCount := 0
		for _, albumEntry := range albumEntries {
			// Пропускаем поддиректории
			if albumEntry.IsDir() {
				continue
			}
			
			// Проверяем, что файл является изображением по расширению
			filename := albumEntry.Name()
			ext := strings.ToLower(filepath.Ext(filename))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
				imageCount++
			}
		}
		
		// Добавляем альбом в список
		albums = append(albums, AlbumInfo{
			ID:         albumID,
			Name:       albumID, // Пока используем ID как имя
			ImageCount: imageCount,
		})
	}
	
	return albums, nil
}

// createAlbum создает новый альбом для пользователя
func createAlbum(userID string) (string, error) {
	// Генерируем 4-символьный hex ID (2 байта)
	bytes := make([]byte, 2)
	_, err := rand.Read(bytes)
	var albumID string
	if err != nil {
		// В случае ошибки генерации случайных данных, используем timestamp
		albumID = fmt.Sprintf("%04x", time.Now().UnixNano()%65536)
	} else {
		albumID = hex.EncodeToString(bytes)
	}
	
	// Создаем директорию для альбома
	err = ensureAlbumDir(userID, albumID)
	if err != nil {
		return "", err
	}
	
	return albumID, nil
}