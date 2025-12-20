package main

import (
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// indexHandler обрабатывает главную страницу и альбомы/изображения
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Если это не главная страница, проверяем альбом/изображение
	if r.URL.Path != "/" {
		contentHandler(w, r)
		return
	}
	
	if r.Method == "GET" {
		// Получаем ID сессии пользователя
		sessionID := getSessionID(w, r)
		
		// Получаем список альбомов пользователя
		albums, err := getUserAlbums(sessionID)
		if err != nil {
			// В случае ошибки продолжаем с пустым списком
			albums = []AlbumInfo{}
		}
		
		// Добавляем логирование для проверки структуры данных
		fmt.Printf("[DEBUG] indexHandler: received %d albums\n", len(albums))
		for i, album := range albums {
			fmt.Printf("[DEBUG]   [%d] ID=%s, CreatedAt=%v\n", i, album.ID, album.CreatedAt)
		}
		
		// Структура для передачи данных в шаблон
		data := struct {
			Albums    []AlbumInfo
			HasAlbums bool
			SessionID string
		}{
			Albums:    albums,
			HasAlbums: len(albums) > 0,
			SessionID: sessionID,
		}
		
		// Отображаем главную страницу
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}
	
	// Для других методов возвращаем ошибку
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// uploadHandler обрабатывает загрузку изображений
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Получаем ID сессии пользователя
	sessionID := getSessionID(w, r)
	
	// Ограничиваем размер запроса до 10MB
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	
	// Получаем album_id из формы
	albumID := r.FormValue("album_id")
	
	// Если album_id не указан, создаем новый альбом автоматически
	if albumID == "" {
		newAlbumID, err := createAlbum(sessionID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creating album: %v", err), http.StatusInternalServerError)
			return
		}
		albumID = newAlbumID
	}
	
	// Проверяем наличие файлов в запросе
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		http.Error(w, "No files in request", http.StatusBadRequest)
		return
	}
	
	// Получаем все файлы из формы
	files := r.MultipartForm.File["image"]
	if len(files) == 0 {
		http.Error(w, "No files selected", http.StatusBadRequest)
		return
	}
	
	// Обрабатываем файлы параллельно
	fmt.Printf("[DEBUG] uploadHandler: начало параллельной обработки %d файлов\n", len(files))
	parallelStartTime := time.Now()
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error
	var processedCount int
	
	// Ограничиваем количество одновременных goroutines для избежания перегрузки
	maxWorkers := 4
	if len(files) < maxWorkers {
		maxWorkers = len(files)
	}
	
	// Канал для задач
	fileChan := make(chan *multipart.FileHeader, len(files))
	
	// Запускаем worker'ы
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for fileHeader := range fileChan {
				startTime := time.Now()
				fmt.Printf("[DEBUG] uploadHandler: worker %d начало обработки файла: %s\n", workerID, fileHeader.Filename)
				
				// Открываем файл
				file, err := fileHeader.Open()
				if err != nil {
					mu.Lock()
					errors = append(errors, fmt.Errorf("worker %d: error opening file %s: %v", workerID, fileHeader.Filename, err))
					mu.Unlock()
					continue
				}
				
				// Сохраняем изображение
				_, err = saveImage(file, fileHeader, sessionID, albumID)
				file.Close()
				
				if err != nil {
					mu.Lock()
					errors = append(errors, fmt.Errorf("worker %d: error saving file %s: %v", workerID, fileHeader.Filename, err))
					mu.Unlock()
					continue
				}
				
				totalTime := time.Since(startTime)
				mu.Lock()
				processedCount++
				fmt.Printf("[INFO] uploadHandler: worker %d сохранил файл %s (время: %v, всего обработано: %d/%d)\n",
					workerID, fileHeader.Filename, totalTime, processedCount, len(files))
				mu.Unlock()
			}
		}(i)
	}
	
	// Отправляем файлы в канал
	for _, fileHeader := range files {
		fileChan <- fileHeader
	}
	close(fileChan)
	
	// Ждем завершения всех worker'ов
	wg.Wait()
	
	parallelTotalTime := time.Since(parallelStartTime)
	fmt.Printf("[DEBUG] uploadHandler: параллельная обработка завершена за %v\n", parallelTotalTime)
	
	// Проверяем наличие ошибок
	if len(errors) > 0 {
		errorMsg := "Errors occurred during upload:\n"
		for _, err := range errors {
			errorMsg += err.Error() + "\n"
		}
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}
	
	fmt.Printf("[INFO] uploadHandler: успешно загружено файлов: %d (общее время: %v)\n", len(files), parallelTotalTime)
	
	// Перенаправляем на страницу альбома
	http.Redirect(w, r, "/"+sessionID+"/"+albumID, http.StatusSeeOther)
}

// contentHandler обрабатывает отдачу изображений или страницы альбома (без префикса)
func contentHandler(w http.ResponseWriter, r *http.Request) {
	// Пропускаем начальный "/"
	path := r.URL.Path[1:]
	if path == "" {
		http.NotFound(w, r)
		return
	}
	
	// Разбираем путь (формат: /{sessionID}/{albumID} или /{sessionID}/{albumID}/{filename})
	parts := strings.SplitN(path, "/", 3)
	
	// Если 2 сегмента - это страница альбома
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		sessionID := parts[0]
		albumID := parts[1]
		
		// Получаем ID сессии текущего пользователя
		currentSessionID := getSessionID(w, r)
		
		// Проверяем, является ли текущий пользователь владельцем альбома
		isOwner := (currentSessionID == sessionID)
		
		// Добавляем логирование для диагностики
		fmt.Printf("[DEBUG] contentHandler: albumID=%s, ownerSessionID=%s, currentSessionID=%s, isOwner=%v\n",
			albumID, sessionID, currentSessionID, isOwner)
		
		// Отображаем страницы альбома
		data := struct {
			Images         []ImageInfo
			HasImages      bool
			SessionID      string
			OwnerSessionID string
			AlbumID        string
			IsOwner        bool
		}{
			SessionID:      currentSessionID,
			OwnerSessionID: sessionID,
			AlbumID:        albumID,
			IsOwner:        isOwner,
		}
		
		// Получаем все изображения альбома (используем sessionID владельца)
		images, err := getUserImagesPaginated(sessionID, albumID, 0, 0)
		if err != nil {
			images = []ImageInfo{}
		}
		data.Images = images
		data.HasImages = len(images) > 0
		
		// Отображаем страницу альбома
		tmpl, err := template.ParseFiles("templates/album.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	
	// Если 3 сегмента - это файл изображения
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		http.NotFound(w, r)
		return
	}
	
	sessionID := parts[0]
	albumID := parts[1]
	filename := parts[2]
	
	// Формируем путь к файлу
	filePath := "data/" + sessionID + "/" + albumID + "/" + filename
	
	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	
	// Отдаем файл
	http.ServeFile(w, r, filePath)
}

// deleteImageHandler обрабатывает удаление изображения
func deleteImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Получаем ID сессии из cookie
	sessionID := getSessionID(w, r)
	
	// Получаем параметры из формы
	albumID := r.FormValue("album_id")
	filename := r.FormValue("filename")
	
	if albumID == "" || filename == "" {
		http.Error(w, "album_id and filename required", http.StatusBadRequest)
		return
	}
	
	// Удаляем изображение
	err := deleteImage(sessionID, albumID, filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting image: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Возвращаем успешный ответ
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Image deleted successfully"))
}

// deleteAlbumHandler обрабатывает удаление альбома
func deleteAlbumHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Получаем ID сессии из cookie
	sessionID := getSessionID(w, r)
	
	// Получаем album_id из формы
	albumID := r.FormValue("album_id")
	
	if albumID == "" {
		http.Error(w, "album_id required", http.StatusBadRequest)
		return
	}
	
	// Удаляем альбом
	err := deleteAlbum(sessionID, albumID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting album: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Перенаправляем на главную страницу
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// deleteUserHandler обрабатывает удаление профиля пользователя
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Получаем ID сессии из cookie
	sessionID := getSessionID(w, r)
	
	// Удаляем все данные пользователя
	err := deleteUser(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting user data: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Очищаем cookie, устанавливая время жизни в прошлом
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	
	// Возвращаем успешный ответ
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Profile deleted successfully"))
}

