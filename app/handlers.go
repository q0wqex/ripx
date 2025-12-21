package main

import (
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"sync"
)

// indexHandler обрабатывает главную страницу
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Если это не главная страница, обрабатываем как контент
	if r.URL.Path != "/" {
		contentHandler(w, r)
		return
	}

	// Получаем сессию пользователя
	sessionID := getSessionID(w, r)

	// Получаем список альбомов
	albums, err := getUserAlbums(sessionID)
	if err != nil {
		albums = []AlbumInfo{}
	}

	// Подготавливаем данные для шаблона
	data := struct {
		Albums    []AlbumInfo
		HasAlbums bool
		SessionID string
	}{
		Albums:    albums,
		HasAlbums: len(albums) > 0,
		SessionID: sessionID,
	}

	// Отображаем страницу
	if err := renderTemplate(w, TemplatesPath+"/index.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// uploadHandler обрабатывает загрузку изображений
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("Начало обработки загрузки")
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := getSessionID(w, r)
	logger.Debug("Session ID: " + sessionID)

	// Ограничиваем размер запроса
	if err := r.ParseMultipartForm(MaxFileSize); err != nil {
		logger.Error("Ошибка парсинга формы: " + err.Error())
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Получаем ID альбома
	albumID := getAlbumID(r, sessionID)
	logger.Debug("Album ID: " + albumID)

	// Проверяем файлы
	files := getUploadFiles(r)
	logger.Debug("Количество файлов для загрузки: " + fmt.Sprintf("%d", len(files)))
	if len(files) == 0 {
		http.Error(w, "No files selected", http.StatusBadRequest)
		return
	}

	// Обрабатываем файлы
	logger.Info("Начало обработки " + fmt.Sprintf("%d", len(files)) + " файлов")
	if err := processUpload(files, sessionID, albumID, w); err != nil {
		logger.Error("Ошибка обработки загрузки: " + err.Error())
		http.Error(w, fmt.Sprintf("Upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Info("Загрузка завершена успешно, перенаправление на альбом")
	// Перенаправляем на альбом
	http.Redirect(w, r, "/"+sessionID+"/"+albumID, http.StatusSeeOther)
}

// contentHandler обрабатывает отдачу изображений или страницы альбома
func contentHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.SplitN(path, "/", 3)
	
	switch len(parts) {
	case 2:
		// Страница альбома
		handleAlbumPage(w, r, parts[0], parts[1])
	case 3:
		// Файл изображения
		handleImageFile(w, r, parts[0], parts[1], parts[2])
	default:
		http.NotFound(w, r)
	}
}

// handleAlbumPage обрабатывает страницу альбома
func handleAlbumPage(w http.ResponseWriter, r *http.Request, sessionID, albumID string) {
	currentSessionID := getSessionID(w, r)
	isOwner := currentSessionID == sessionID

	images, err := getUserImagesPaginated(sessionID, albumID, 0, 0)
	if err != nil {
		images = []ImageInfo{}
	}

	data := struct {
		Images         []ImageInfo
		HasImages      bool
		SessionID      string
		OwnerSessionID string
		AlbumID        string
		IsOwner        bool
	}{
		Images:         images,
		HasImages:      len(images) > 0,
		SessionID:      currentSessionID,
		OwnerSessionID: sessionID,
		AlbumID:        albumID,
		IsOwner:        isOwner,
	}

	if err := renderTemplate(w, TemplatesPath+"/album.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleImageFile обрабатывает отдачу файла изображения
func handleImageFile(w http.ResponseWriter, r *http.Request, sessionID, albumID, filename string) {
	filePath := DataPath + "/" + sessionID + "/" + albumID + "/" + filename

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filePath)
}

// deleteImageHandler обрабатывает удаление изображения
func deleteImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := getSessionID(w, r)
	albumID := r.FormValue("album_id")
	filename := r.FormValue("filename")

	if albumID == "" || filename == "" {
		http.Error(w, "album_id and filename required", http.StatusBadRequest)
		return
	}

	if err := deleteImage(sessionID, albumID, filename); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting image: %v", err), http.StatusInternalServerError)
		return
	}

	SuccessResponse(w, map[string]string{"message": "Image deleted successfully"})
}

// deleteAlbumHandler обрабатывает удаление альбома
func deleteAlbumHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := getSessionID(w, r)
	albumID := r.FormValue("album_id")

	if albumID == "" {
		http.Error(w, "album_id required", http.StatusBadRequest)
		return
	}

	if err := deleteAlbum(sessionID, albumID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting album: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// deleteUserHandler обрабатывает удаление профиля пользователя
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := getSessionID(w, r)

	if err := deleteUser(sessionID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting user data: %v", err), http.StatusInternalServerError)
		return
	}

	// Очищаем cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	SuccessResponse(w, map[string]string{"message": "Profile deleted successfully"})
}

// Вспомогательные функции

// getAlbumID получает или создает ID альбома
func getAlbumID(r *http.Request, sessionID string) string {
	albumID := r.FormValue("album_id")
	if albumID != "" {
		return albumID
	}

	// Создаем новый альбом если не указан
	newAlbumID, err := createAlbum(sessionID)
	if err != nil {
		return ""
	}
	return newAlbumID
}

// getUploadFiles извлекает файлы из запроса
func getUploadFiles(r *http.Request) []*multipart.FileHeader {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil
	}

	files := r.MultipartForm.File["image"]
	if len(files) == 0 {
		return nil
	}

	return files
}

// processUpload обрабатывает загрузку файлов
func processUpload(files []*multipart.FileHeader, sessionID, albumID string, w http.ResponseWriter) error {
	// Для небольшого количества файлов используем последовательную обработку
	if len(files) <= 5 {
		return processSequentialUpload(files, sessionID, albumID)
	}

	// Для большого количества файлов используем параллельную обработку
	return processParallelUpload(files, sessionID, albumID)
}

// processSequentialUpload обрабатывает файлы последовательно
func processSequentialUpload(files []*multipart.FileHeader, sessionID, albumID string) error {
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return fmt.Errorf("error opening file %s: %v", fileHeader.Filename, err)
		}
		defer file.Close()

		_, err = saveImage(file, fileHeader, sessionID, albumID)
		if err != nil {
			return fmt.Errorf("error saving file %s: %v", fileHeader.Filename, err)
		}
	}
	return nil
}

// processParallelUpload обрабатывает файлы параллельно
func processParallelUpload(files []*multipart.FileHeader, sessionID, albumID string) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	workerCount := MaxWorkers
	if len(files) < workerCount {
		workerCount = len(files)
	}

	fileChan := make(chan *multipart.FileHeader, len(files))

	// Запускаем воркеров
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for fileHeader := range fileChan {
				if err := processSingleFile(fileHeader, sessionID, albumID); err != nil {
					mu.Lock()
					errors = append(errors, fmt.Errorf("worker %d: %v", workerID, err))
					mu.Unlock()
				}
			}
		}(i)
	}

	// Отправляем файлы на обработку
	for _, fileHeader := range files {
		fileChan <- fileHeader
	}
	close(fileChan)

	// Ждем завершения
	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("upload errors: %v", errors)
	}

	return nil
}

// processSingleFile обрабатывает один файл
func processSingleFile(fileHeader *multipart.FileHeader, sessionID, albumID string) error {
	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("error opening file %s: %v", fileHeader.Filename, err)
	}
	defer file.Close()

	_, err = saveImage(file, fileHeader, sessionID, albumID)
	if err != nil {
		return fmt.Errorf("error saving file %s: %v", fileHeader.Filename, err)
	}

	return nil
}

// renderTemplate рендерит HTML шаблон
func renderTemplate(w http.ResponseWriter, templatePath string, data interface{}) error {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}