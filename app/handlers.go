package main

import (
	"fmt"
	"html/template"
	"net/http"
)

// indexHandler обрабатывает главную страницу
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	if r.Method == "GET" {
		// Отображаем главную страницу
		tmpl, err := template.ParseFiles("app/templates/index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
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
	r.ParseMultipartForm(10 << 20) // 10 MB
	
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
	
	// Отображаем страницу альбома
	tmpl, err := template.ParseFiles("app/templates/album.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// deleteHandler обрабатывает удаление изображений
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	// Реализация будет добавлена в следующей фазе
}