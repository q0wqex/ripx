package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Path builders for data directories
func userPath(userID string) string           { return filepath.Join(DataPath, userID) }
func albumPath(userID, albumID string) string { return filepath.Join(DataPath, userID, albumID) }
func imagePath(userID, albumID, filename string) string {
	return filepath.Join(DataPath, userID, albumID, filename)
}

// Глобальная переменная для хранения общего количества изображений
var TotalImageCount int

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
	albumPath := albumPath(userID, albumID)
	if err := EnsureDir(albumPath); err != nil {
		return nil, err
	}

	// Генерация уникального имени файла
	filename := generateUniqueFilename(extension)
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

	// Увеличиваем глобальный счетчик изображений
	TotalImageCount++

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
func generateUniqueFilename(extension string) string {
	ext := strings.ToLower(extension)
	if ext == "" {
		ext = "webp" // расширение по умолчанию
	} else if !strings.HasPrefix(ext, ".") {
		ext = "." + ext // добавляем точку если её нет
	}

	randomID := RandomID()
	return randomID + ext
}

// getUserImages возвращает список изображений пользователя
func getUserImages(userID, albumID string) ([]ImageInfo, error) {
	dirPath := albumPath(userID, albumID)

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
			Path:     filepath.Join(dirPath, filename),
			Size:     info.Size(),
			UserID:   userID,
			AlbumID:  albumID,
		})
	}

	// Сортировка изображений по времени модификации (старые сверху, новые снизу)
	sort.Slice(images, func(i, j int) bool {
		infoI, errI := os.Stat(images[i].Path)
		infoJ, errJ := os.Stat(images[j].Path)

		if errI != nil || errJ != nil {
			return false // если не можем получить статус файла, не меняем порядок
		}

		return infoI.ModTime().Before(infoJ.ModTime())
	})

	return images, nil
}

// getSessionID получает или генерирует ID сессии пользователя
func getSessionID(w http.ResponseWriter, r *http.Request) string {
	// Проверка наличия cookie
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil && cookie.Value != "" {
		logger.Debug(fmt.Sprintf("getSessionID: using existing cookie, sessionID=%s", cookie.Value))
		return cookie.Value
	}

	// Генерация нового ID сессии
	sessionID := RandomID()
	logger.Debug(fmt.Sprintf("getSessionID: creating new session, sessionID=%s", sessionID))

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
	userDir := userPath(userID)
	logger.Debug(fmt.Sprintf("getUserAlbums: userDir=%s", userDir))

	// Проверка существования директории
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		logger.Debug("getUserAlbums: user dir does not exist")
		return []AlbumInfo{}, nil
	}

	// Чтение содержимого директории
	entries, err := os.ReadDir(userDir)
	if err != nil {
		logger.Debug(fmt.Sprintf("getUserAlbums: error reading dir: %v", err))
		return nil, err
	}
	logger.Debug(fmt.Sprintf("getUserAlbums: found %d entries", len(entries)))

	var albums []AlbumInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		albumID := entry.Name()
		albumDir := filepath.Join(userDir, albumID)

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
	count := 0
	processDir(dirPath, func(entry os.DirEntry) bool {
		return !entry.IsDir() && IsImageFile(entry.Name())
	}, func(path string, info os.FileInfo) error {
		count++
		return nil
	})
	return count
}

// createAlbum создает новый альбом для пользователя
func createAlbum(userID string) (string, error) {
	albumID := RandomID()

	// Создание директории для альбома
	albumDir := albumPath(userID, albumID)
	logger.Debug(fmt.Sprintf("createAlbum: creating albumDir=%s", albumDir))
	if err := EnsureDir(albumDir); err != nil {
		return "", err
	}
	logger.Debug(fmt.Sprintf("createAlbum: album created, albumID=%s", albumID))

	return albumID, nil
}

// deleteImage удаляет изображение
func deleteImage(userID, albumID, filename string) error {
	filePath := imagePath(userID, albumID, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("image not found")
	}

	err := os.Remove(filePath)
	if err == nil {
		// Уменьшаем глобальный счетчик изображений
		TotalImageCount--
	}
	return err
}

// deleteAlbum удаляет альбом со всеми изображениями
func deleteAlbum(userID, albumID string) error {
	albumDir := albumPath(userID, albumID)

	if _, err := os.Stat(albumDir); os.IsNotExist(err) {
		return fmt.Errorf("album not found")
	}

	// Подсчитываем количество изображений в альбоме перед удалением
	imageCount := countImagesInDir(albumDir)

	err := os.RemoveAll(albumDir)
	if err == nil {
		// Уменьшаем глобальный счетчик изображений на количество удаленных изображений
		TotalImageCount -= imageCount
	}
	return err
}

// deleteUser удаляет все данные пользователя
func deleteUser(userID string) error {
	userDir := userPath(userID)

	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		return fmt.Errorf("user directory not found")
	}

	// Подсчитываем количество изображений в пользовательской директории перед удалением
	totalImages := 0
	// Рекурсивный обход всей директории пользователя
	err := filepath.Walk(userDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Пропускаем ошибки доступа к файлам
			return nil
		}

		// Пропускаем директории
		if !info.IsDir() {
			// Проверяем, является ли файл изображением
			if IsImageFile(info.Name()) {
				totalImages++
			}
		}

		return nil
	})

	errRemove := os.RemoveAll(userDir)
	if errRemove == nil && err == nil {
		// Уменьшаем глобальный счетчик изображений на количество удаленных изображений
		TotalImageCount -= totalImages
	}
	return errRemove
}

// countAllFilesInDataPath подсчитывает количество всех файлов в директории data при запуске приложения
func countAllFilesInDataPath() int {
	count := 0

	// Рекурсивный обход всей директории DataPath
	err := filepath.Walk(DataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Пропускаем ошибки доступа к файлам
			return nil
		}

		// Пропускаем директории
		if !info.IsDir() {
			count++
		}

		return nil
	})

	if err != nil {
		logger.Error(fmt.Sprintf("countAllFilesInDataPath: error walking data directory: %v", err))
	}

	return count
}

// convertedPath возвращает путь к директории конвертированных файлов
func convertedPath(userID string) string {
	return filepath.Join(DataPath, userID, "converted")
}

// validateVideoType проверяет тип видео файла
func validateVideoType(file multipart.File) (string, bool) {
	// Чтение заголовка файла
	buffer := make([]byte, 512)
	if _, err := file.Read(buffer); err != nil {
		return "", false
	}

	// Восстановление указателя
	file.Seek(0, 0)

	// Определение MIME типа
	contentType := http.DetectContentType(buffer)

	// Проверка на MP4 видео
	if contentType == "video/mp4" {
		return "mp4", true
	}

	return "", false
}

// convertMp4ToGif конвертирует MP4 файл в GIF используя FFmpeg
func convertMp4ToGif(inputPath, outputPath string) error {
	// Проверка наличия FFmpeg в системе
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("FFmpeg not found in system: %v", err)
	}

	// Команда FFmpeg для конвертации:
	// -i: входной файл
	// -vf: фильтр видео (fps=10 - 10 кадров в секунду, scale=320:-1 - ширина 320px, высота авто)
	// -c:v gif: кодек для GIF
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", "fps=30,scale=720:-1:flags=lanczos",
		"-c:v", "gif",
		outputPath,
	)

	// Перенаправляем stderr для логирования
	cmd.Stderr = os.Stderr

	logger.Debug(fmt.Sprintf("convertMp4ToGif: executing ffmpeg command: %v", cmd.Args))

	// Запуск команды
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FFmpeg conversion failed: %v", err)
	}

	logger.Debug(fmt.Sprintf("convertMp4ToGif: successfully converted %s to %s", inputPath, outputPath))
	return nil
}

// saveConvertedGif сохраняет конвертированный GIF файл
func saveConvertedGif(inputFile multipart.File, header *multipart.FileHeader, userID string) (string, error) {
	// Проверка размера файла
	if header.Size > MaxVideoSize {
		return "", fmt.Errorf("video file too large: %d bytes (max %d)", header.Size, MaxVideoSize)
	}

	// Валидация типа видео
	extension, valid := validateVideoType(inputFile)
	if !valid {
		return "", fmt.Errorf("invalid video type, only MP4 is supported")
	}

	// Создание директории для конвертированных файлов
	convertedDir := convertedPath(userID)
	if err := EnsureDir(convertedDir); err != nil {
		return "", err
	}

	// Генерация уникального имени файла для временного MP4
	tempFilename := generateUniqueFilename(extension)
	tempInputPath := filepath.Join(convertedDir, tempFilename)

	// Создание временного файла MP4
	dst, err := os.Create(tempInputPath)
	if err != nil {
		return "", err
	}
	defer os.Remove(tempInputPath) // Удаляем временный файл после конвертации
	defer dst.Close()

	// Копирование содержимого
	if _, err := io.Copy(dst, inputFile); err != nil {
		return "", err
	}

	// Генерация имени для выходного GIF файла
	gifFilename := generateUniqueFilename("gif")
	outputPath := filepath.Join(convertedDir, gifFilename)

	// Конвертация MP4 в GIF
	if err := convertMp4ToGif(tempInputPath, outputPath); err != nil {
		return "", err
	}

	// Проверка существования выходного файла
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("GIF file was not created")
	}

	logger.Debug(fmt.Sprintf("saveConvertedGif: GIF saved to %s", outputPath))

	// Возвращаем путь к GIF файлу относительно DataPath
	return filepath.Join(userID, "converted", gifFilename), nil
}
