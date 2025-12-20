package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
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
	
	// Обрабатываем каждый файл
	for i, fileHeader := range files {
		// Открываем файл
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error opening file %s: %v", fileHeader.Filename, err), http.StatusInternalServerError)
			return
		}
		
		// Сохраняем изображение
		_, err = saveImage(file, fileHeader, sessionID, albumID)
		file.Close()
		
		if err != nil {
			http.Error(w, fmt.Sprintf("Error saving image %s: %v", fileHeader.Filename, err), http.StatusInternalServerError)
			return
		}
		
		fmt.Printf("[INFO] uploadHandler: сохранен файл %d/%d: %s\n", i+1, len(files), fileHeader.Filename)
	}
	
	fmt.Printf("[INFO] uploadHandler: успешно загружено файлов: %d\n", len(files))
	
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
		
		// Отображаем страницу альбома
		data := struct {
			Images    []ImageInfo
			HasImages bool
			SessionID string
			AlbumID   string
		}{
			SessionID: sessionID,
			AlbumID:   albumID,
		}
		
		// Получаем все изображения альбома
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
	filePath := "/data/" + sessionID + "/" + albumID + "/" + filename
	
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
	fmt.Fprintf(w, "Image deleted successfully")
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

