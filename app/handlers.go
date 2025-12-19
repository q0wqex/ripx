package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
)

// indexHandler обрабатывает главную страницу
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	if r.Method == "GET" {
		// Отображаем главную страницу
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
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
	
	// Получаем файл из формы
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	// Сохраняем изображение
	_, err = saveImage(file, header, sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error saving image: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Перенаправляем на страницу альбома
	http.Redirect(w, r, "/album", http.StatusSeeOther)
}

// albumHandler обрабатывает страницу альбома
func albumHandler(w http.ResponseWriter, r *http.Request) {
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
	
	log.Printf("[DEBUG albumHandler] SessionID: %s (cookie err: %v)", sessionID, err)
	
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
	}{
		CurrentPage: page,
		TotalPages: 0,
		HasPagination: false,
	}
	
	// Получаем список изображений пользователя
	if sessionID != "" {
		// Сначала получаем все изображения для подсчета общего количества
		allImages, err := getUserImages(sessionID)
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
			data.Images, err = getUserImagesPaginated(sessionID, page, pageSize)
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
	
	// Получаем имя файла из URL
	filename := r.URL.Path[len("/image/"):]
	if filename == "" {
		log.Printf("[DEBUG imageHandler] Пустое имя файла")
		http.NotFound(w, r)
		return
	}
	
	log.Printf("[DEBUG imageHandler] Запрос изображения: filename=%s", filename)
	
	// Получаем ID сессии из cookie
	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Printf("[DEBUG imageHandler] Cookie отсутствует или ошибка: %v", err)
		http.NotFound(w, r)
		return
	}
	sessionID := cookie.Value
	
	log.Printf("[DEBUG imageHandler] SessionID из cookie: %s", sessionID)
	
	// Формируем путь к файлу
	filePath := "/data/" + sessionID + "/" + filename
	
	log.Printf("[DEBUG imageHandler] Путь к файлу: %s", filePath)
	
	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("[DEBUG imageHandler] Файл не найден: %s", filePath)
		http.NotFound(w, r)
		return
	}
	
	log.Printf("[DEBUG imageHandler] Файл найден, отправляем клиенту")
	
	// Отдаем файл
	http.ServeFile(w, r, filePath)
}
