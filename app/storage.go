package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

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
	CreatedAt  time.Time
}

// saveImage сохраняет загруженное изображение
func saveImage(file multipart.File, header *multipart.FileHeader, userID, albumID string) (*ImageInfo, error) {
	// Проверка размера файла
	if header.Size > MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes", header.Size)
	}

	// Валидация типа изображения
	extension, valid := validateImageType(file)
	if !valid {
		return nil, fmt.Errorf("invalid image type")
	}

	// Создание директории для альбома
	albumPath := DataPath + "/" + userID + "/" + albumID
	if err := EnsureDir(albumPath); err != nil {
		return nil, err
	}

	// Генерация уникального имени файла
	filename := generateUniqueFilename(header.Filename, extension)
	filePath := albumPath + "/" + filename

	// Создание файла
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	// Копирование содержимого
	if _, err := io.Copy(dst, file); err != nil {
		return nil, err
	}

	// Получение информации о файле
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	return &ImageInfo{
		Filename: filename,
		Path:     filePath,
		Size:     stat.Size(),
		UserID:   userID,
		AlbumID:  albumID,
	}, nil
}

// validateImageType проверяет тип изображения
func validateImageType(file multipart.File) (string, bool) {
	// Чтение заголовка файла
	buffer := make([]byte, 512)
	if _, err := file.Read(buffer); err != nil {
		return "", false
	}

	// Восстановление указателя
	file.Seek(0, 0)

	// Определение MIME типа
	contentType := http.DetectContentType(buffer)

	// Проверка разрешенных типов
	if !AllowedImageTypes[contentType] {
		return "", false
	}

	// Возвращаем соответствующее расширение
	if ext, exists := ImageExtensions[contentType]; exists {
		return ext, true
	}

	return "", false
}

// generateUniqueFilename генерирует уникальное имя файла
func generateUniqueFilename(originalFilename, extension string) string {
	ext := strings.ToLower(extension)
	if ext == "" {
		ext = ".jpg" // расширение по умолчанию
	} else if !strings.HasPrefix(ext, ".") {
		ext = "." + ext // добавляем точку если её нет
	}

	randomID := RandomID()
	return randomID + ext
}

// getUserImages возвращает список изображений пользователя
func getUserImages(userID, albumID string) ([]ImageInfo, error) {
	dirPath := DataPath + "/" + userID
	if albumID != "" {
		dirPath += "/" + albumID
	}

	// Проверка существования директории
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return []ImageInfo{}, nil
	}

	// Чтение содержимого директории
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var images []ImageInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !IsImageFile(filename) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		images = append(images, ImageInfo{
			Filename: filename,
			Path:     dirPath + "/" + filename,
			Size:     info.Size(),
			UserID:   userID,
			AlbumID:  albumID,
		})
	}



	return images, nil
}

// getSessionID получает или генерирует ID сессии пользователя
func getSessionID(w http.ResponseWriter, r *http.Request) string {
	// Проверка наличия cookie
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Генерация нового ID сессии
	sessionID := RandomID()

	// Установка cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   SessionMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return sessionID
}

// getUserAlbums возвращает список альбомов пользователя
func getUserAlbums(userID string) ([]AlbumInfo, error) {
	userDir := DataPath + "/" + userID

	// Проверка существования директории
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		return []AlbumInfo{}, nil
	}

	// Чтение содержимого директории
	entries, err := os.ReadDir(userDir)
	if err != nil {
		return nil, err
	}

	var albums []AlbumInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		albumID := entry.Name()
		albumDir := userDir + "/" + albumID

		// Получение информации о директории
		dirInfo, err := os.Stat(albumDir)
		var createdAt time.Time
		if err == nil {
			createdAt = dirInfo.ModTime()
		}

		// Подсчет количества изображений
		imageCount := countImagesInDir(albumDir)

		// Добавление альбома в список
		albums = append(albums, AlbumInfo{
			ID:         albumID,
			Name:       albumID,
			ImageCount: imageCount,
			CreatedAt:  createdAt,
		})
	}

	// Сортировка альбомов по дате создания (новые сверху)
	sort.Slice(albums, func(i, j int) bool {
		if albums[i].CreatedAt.Equal(albums[j].CreatedAt) {
			return albums[i].ID > albums[j].ID
		}
		return albums[i].CreatedAt.After(albums[j].CreatedAt)
	})

	return albums, nil
}

// countImagesInDir подсчитывает количество изображений в директории
func countImagesInDir(dirPath string) int {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && IsImageFile(entry.Name()) {
			count++
		}
	}

	return count
}

// createAlbum создает новый альбом для пользователя
func createAlbum(userID string) (string, error) {
	albumID := RandomID()

	// Создание директории для альбома
	albumPath := DataPath + "/" + userID + "/" + albumID
	if err := EnsureDir(albumPath); err != nil {
		return "", err
	}

	return albumID, nil
}

// deleteImage удаляет изображение
func deleteImage(userID, albumID, filename string) error {
	filePath := DataPath + "/" + userID + "/" + albumID + "/" + filename

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("image not found")
	}

	return os.Remove(filePath)
}

// deleteAlbum удаляет альбом со всеми изображениями
func deleteAlbum(userID, albumID string) error {
	albumPath := DataPath + "/" + userID + "/" + albumID

	if _, err := os.Stat(albumPath); os.IsNotExist(err) {
		return fmt.Errorf("album not found")
	}

	return os.RemoveAll(albumPath)
}

// deleteUser удаляет все данные пользователя
func deleteUser(userID string) error {
	userDir := DataPath + "/" + userID

	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		return fmt.Errorf("user directory not found")
	}

	return os.RemoveAll(userDir)
}