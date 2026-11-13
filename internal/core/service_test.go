package core

import (
	"strings"
	"testing"
)

func TestBcryptPasswordHashing(t *testing.T) {
	rawPassword := "secret_baker_123"

	// 1. Хеширование
	hash, err := HashPassword(rawPassword)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	if !strings.HasPrefix(hash, "$2a$") {
		t.Errorf("expected bcrypt prefix $2a$, got %s", hash)
	}

	// 2. Проверка корректного пароля
	if !CheckPasswordHash(rawPassword, hash) {
		t.Errorf("expected password to match hash")
	}

	// 3. Проверка неверного пароля
	if CheckPasswordHash("wrong_password", hash) {
		t.Errorf("expected wrong password to fail check")
	}
}

func TestUserVerifyPassword(t *testing.T) {
	rawPassword := "pass123"
	hashedPassword, err := HashPassword(rawPassword)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	t.Run("Verify against bcrypt hash", func(t *testing.T) {
		user := &User{PasswordHash: hashedPassword}
		if !user.VerifyPassword("pass123") {
			t.Errorf("expected 'pass123' to verify successfully")
		}
		if user.VerifyPassword("wrong") {
			t.Errorf("expected 'wrong' to fail")
		}
	})

	t.Run("Verify against legacy plaintext (backward compatibility)", func(t *testing.T) {
		legacyUser := &User{PasswordHash: "plain_text_pass"}
		if !legacyUser.VerifyPassword("plain_text_pass") {
			t.Errorf("expected legacy plaintext password to verify successfully")
		}
		if legacyUser.VerifyPassword("another") {
			t.Errorf("expected incorrect password to fail on legacy user")
		}
	})
}
