package main

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"

	"github.com/yourusername/trivia-api/internal/config"
	"github.com/yourusername/trivia-api/internal/domain/entity"
	"github.com/yourusername/trivia-api/pkg/database"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Подключаемся к базе данных
	db, err := database.NewPostgresDB(cfg.Database.PostgresConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Новый пароль для администратора
	newPassword := "12345678" // Поменяй на свой

	// Проверяем, существует ли уже администратор (ID = 1)
	var existingUser entity.User
	result := db.First(&existingUser, 1)

	if result.Error == nil {
		fmt.Println("Администратор уже существует, сбрасываем пароль...")

		// Хешируем новый пароль
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Ошибка при хешировании пароля:", err)
		}

		// Обновляем пароль в БД
		existingUser.Password = string(hashedPassword)
		db.Save(&existingUser)

		fmt.Println("✅ Пароль успешно изменён! Новый пароль:", newPassword)
		return
	}

	// Создаём нового администратора, если его нет
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)

	admin := &entity.User{
		ID:       1,
		Username: "admin",
		Email:    "admin@example.com",
		Password: string(hashedPassword),
	}

	if err := db.Create(admin).Error; err != nil {
		log.Fatalf("Failed to create admin: %v", err)
	}

	fmt.Println("✅ Администратор успешно создан:")
	fmt.Printf("ID: %d, Username: %s, Email: %s\n", admin.ID, admin.Username, admin.Email)
	fmt.Println("Пароль:", newPassword)
}
