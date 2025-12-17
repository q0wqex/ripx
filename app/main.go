package main

import (
	"net/http"
)

func main() {
	// Создание директории для хранения данных
	err := ensureDataDir()
	if err != nil {
		return
	}
	
	// Настройка маршрутов
	mux := http.NewServeMux()
	
	// Регистрация обработчиков
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/upload", uploadHandler)
	mux.HandleFunc("/album", albumHandler)
	
	// Запуск сервера
	http.ListenAndServe("0.0.0.0:8000", mux)
}