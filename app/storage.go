package main

import (
	"mime/multipart"
	"os"
)

// ImageInfo хранит информацию об изображении
type ImageInfo struct {
	Filename string
	Path     string
	Size     int64
	UserID   string
}

// saveImage сохраняет загруженное изображение
func saveImage(file multipart.File, header *multipart.FileHeader, userID string) (*ImageInfo, error) {
	// Реализация будет добавлена в следующей фазе
	return nil, nil
}

// getUserImages возвращает список изображений пользователя
func getUserImages(userID string) ([]ImageInfo, error) {
	// Реализация будет добавлена в следующей фазе
	return nil, nil
}

// deleteImage удаляет изображение
func deleteImage(filename, userID string) error {
	// Реализация будет добавлена в следующей фазе
	return nil
}

// ensureDataDir создает директорию /data если она не существует
func ensureDataDir() error {
	// Реализация будет добавлена в следующей фазе
	return os.MkdirAll("/data", 0755)
}