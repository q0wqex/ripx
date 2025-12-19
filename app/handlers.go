package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// indexHandler обрабатывает главную страницу
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
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
	http.Redirect(w, r, "/album?id="+albumID, http.StatusSeeOther)
}

// albumHandler обрабатывает страницу альбома
func albumHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[DEBUG] albumHandler: request received for URL=%s\n", r.URL.String())
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Получаем ID сессии из cookie
	cookie, err := r.Cookie("session_id")
	var sessionID string
	if err == nil && cookie.Value != "" {
		sessionID = cookie.Value
	}
	fmt.Printf("[DEBUG] albumHandler: sessionID = %s\n", sessionID)
	
	// Получаем album_id из query параметра
	albumID := r.URL.Query().Get("id")
	if albumID == "" {
		fmt.Printf("[DEBUG] albumHandler: album_id is empty\n")
		http.Error(w, "album_id required", http.StatusBadRequest)
		return
	}
	fmt.Printf("[DEBUG] albumHandler: albumID = %s\n", albumID)
	
	// Получаем параметры пагинации из URL
	page := 0
	pageSize := 12 // Фиксированный размер страницы
	
	// Парсим параметр page из URL
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p >= 0 {
			page = p
		}
	}
	
	// Структура для передачи данных в шаблон
	data := struct {
		Images []ImageInfo
		HasImages bool
		CurrentPage int
		TotalPages int
		HasPagination bool
		SessionID string
		AlbumID string
	}{
		CurrentPage: page,
		TotalPages: 0,
		HasPagination: false,
		SessionID: sessionID,
		AlbumID: albumID,
	}
	
	// Получаем список изображений пользователя
	if sessionID != "" {
		// Сначала получаем все изображения для подсчета общего количества
		allImages, err := getUserImagesPaginated(sessionID, albumID, 0, 0)
		if err != nil {
			// В случае ошибки, просто продолжаем с пустым списком
			allImages = []ImageInfo{}
		}
		
		// Вычисляем общее количество страниц
		if len(allImages) > 0 {
			data.TotalPages = (len(allImages) + pageSize - 1) / pageSize
			data.HasPagination = data.TotalPages > 1
			
			// Проверяем, что номер страницы не превышает общее количество страниц
			if page >= data.TotalPages {
				page = data.TotalPages - 1
				if page < 0 {
					page = 0
				}
			}
			data.CurrentPage = page
			
			// Получаем изображения для текущей страницы
			data.Images, err = getUserImagesPaginated(sessionID, albumID, page, pageSize)
			if err != nil {
				data.Images = []ImageInfo{}
			}
		}
		
		data.HasImages = len(data.Images) > 0
	}
	
	// Создаем шаблон с функциями
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"iterate": func(count int) []int {
			var items []int
			for i := 0; i < count; i++ {
				items = append(items, i)
			}
			return items
		},
	}
	
	// Отображаем страницу альбома
	tmpl := template.New("album.html").Funcs(funcMap)
	tmpl, err = tmpl.ParseFiles("templates/album.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// imageHandler обрабатывает отдачу изображений
func imageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Получаем путь из URL (формат: /image/{sessionID}/{albumID}/{filename})
	path := r.URL.Path[len("/image/"):]
	if path == "" {
		http.NotFound(w, r)
		return
	}
	
	// Разбираем путь на sessionID, albumID и filename
	parts := strings.SplitN(path, "/", 3)
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

