package main

import (
	"log"
	"net/http"
)

func main() {
	// Инициализация сервера
	log.Println("Starting server on :8080")
	
	// Настройка маршрутов
	setupRoutes()
	
	// Запуск сервера
	err := http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		log.Fatal("Server error: ", err)
	}
}

func setupRoutes() {
	// Настройка маршрутов будет реализована в следующей фазе
}