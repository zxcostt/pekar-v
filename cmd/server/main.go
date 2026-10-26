package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"pekar-v/internal/database"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Строка подключения к нашему контейнеру PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://pekar_user:pekar_password@localhost:5433/pekar_v_db?sslmode=disable"
	}

	// Подключаемся к БД
	dbPool, err := database.NewPostgresPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer dbPool.Close()
	fmt.Println("Успешное подключение к PostgreSQL!")

	// Инициализируем роутер Chi
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Проверка статуса сервера и БД
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "db": "connected"}`))
	})

	// Тестовый роут: выгрузка сотрудников из таблицы users
	r.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
		rows, err := dbPool.Query(r.Context(), "SELECT id, full_name, login, role FROM users")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type UserDTO struct {
			ID       int    `json:"id"`
			FullName string `json:"fullName"`
			Login    string `json:"login"`
			Role     string `json:"role"`
		}

		var users []UserDTO
		for rows.Next() {
			var u UserDTO
			if err := rows.Scan(&u.ID, &u.FullName, &u.Login, &u.Role); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			users = append(users, u)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})

	fmt.Println("HTTP-сервер запущен на http://localhost:8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
