package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Создание директории для хранения данных
	err := ensureDataDir()
	if err != nil {
		os.Exit(1)
	}
	
	// Создаем контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Настройка маршрутов
	mux := http.NewServeMux()
	
	// Регистрация обработчиков
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/upload", uploadHandler)
	mux.HandleFunc("/delete-image", deleteImageHandler)
	mux.HandleFunc("/delete-album", deleteAlbumHandler)
	
	// Запуск cleanup worker в отдельной goroutine
	go startCleanupWorker(ctx)
	
	// Настройка graceful shutdown через сигналы
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Запуск сервера в отдельной goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- http.ListenAndServe("0.0.0.0:8000", mux)
	}()
	
	// Ожидание сигнала или ошибки сервера
	select {
	case <-sigChan:
		// Получен сигнал завершения
		cancel() // Останавливаем cleanup worker
	case <-serverErr:
		// Ошибка сервера
		cancel() // Останавливаем cleanup worker
	}
}