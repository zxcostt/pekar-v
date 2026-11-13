package core

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

const MaxLoginAttempts = 3

// HashPassword хеширует открытый пароль с использованием алгоритма bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPasswordHash сравнивает открытый пароль с bcrypt-хешем
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// User — базовая структура сотрудника (соответствует классу User)
type User struct {
	ID              int     `json:"id"`
	FullName        string  `json:"fullName"`
	Login           string  `json:"login"`
	PasswordHash    string  `json:"-"` // Не отдаем пароль в JSON
	Role            string  `json:"role"`
	IsActive        bool    `json:"isActive"`
	RegisterNumber  *int    `json:"registerNumber,omitempty"`
	WorkshopStation *string `json:"workshopStation,omitempty"`
	WarehouseZoneID *int    `json:"warehouseZoneId,omitempty"`
}

func (u *User) VerifyPassword(inputPassword string) bool {
	// 1. Проверяем bcrypt-хеш
	if CheckPasswordHash(inputPassword, u.PasswordHash) {
		return true
	}
	// 2. Обратная совместимость для старых незахешированных паролей
	return u.PasswordHash == inputPassword
}

func (u *User) GetFullName() string {
	return u.FullName
}

func (u *User) GetRolePermissions() string {
	switch u.Role {
	case "CASHIER":
		return "POS_ACCESS,SHIFT_REPORT,REFUND"
	case "BAKER":
		return "PRODUCTION_ORDER,VIEW_RECIPES"
	case "STOREKEEPER":
		return "WAREHOUSE_SUPPLY,WAREHOUSE_WRITEOFF"
	case "MANAGER":
		return "FULL_ACCESS,MANAGE_RECIPES,SHIFT_REPORT"
	default:
		return "NONE"
	}
}

// Cashier — специализированная модель кассира
type Cashier struct {
	User
	RegisterNumber int `json:"registerNumber"`
}

func (c *Cashier) OpenShift(workDate time.Time) bool {
	return c.IsActive && c.RegisterNumber > 0
}

func (c *Cashier) CloseShift() {}

// Baker — специализированная модель пекаря
type Baker struct {
	User
	WorkshopStation string `json:"workshopStation"`
}

func (b *Baker) ConfirmShiftSchedule() {}

// Storekeeper — специализированная модель кладовщика
type Storekeeper struct {
	User
	WarehouseZoneID int `json:"warehouseZoneId"`
}

func (s *Storekeeper) AuditInventoryBalance(zoneID int) bool {
	return s.IsActive && s.WarehouseZoneID == zoneID
}

// DTO для запроса и ответа авторизации (UC 1)
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User        User   `json:"user"`
	Permissions string `json:"permissions"`
	Message     string `json:"message"`
}

// CreateUserRequest — DTO для создания нового сотрудника управляющим
type CreateUserRequest struct {
	FullName        string  `json:"fullName"`
	Login           string  `json:"login"`
	Password        string  `json:"password"`
	Role            string  `json:"role"`
	RegisterNumber  *int    `json:"registerNumber,omitempty"`
	WorkshopStation *string `json:"workshopStation,omitempty"`
	WarehouseZoneID *int    `json:"warehouseZoneId,omitempty"`
}
