package core

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidCredentials = errors.New("Неверный логин или пароль. Повторите попытку")
	ErrAccountBlocked     = errors.New("Учетная запись заблокирована. Обратитесь к управляющему пекарни")
	ErrUserAlreadyExists  = errors.New("Сотрудник с таким логином уже существует в системе")
	ErrInvalidUserData    = errors.New("Логин, пароль и корректная роль обязательны для заполнения")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Authenticate(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.repo.GetByLogin(ctx, req.Login)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Альтернативная последовательность 1: Неверные учетные данные
	if !user.VerifyPassword(req.Password) {
		return nil, ErrInvalidCredentials
	}

	// Альтернативная последовательность 2: Учетная запись заблокирована
	if !user.IsActive {
		return nil, ErrAccountBlocked
	}

	// Главная последовательность: успешный вход и определение прав роли
	return &LoginResponse{
		User:        *user,
		Permissions: user.GetRolePermissions(),
		Message:     "Авторизация успешна",
	}, nil
}

func (s *Service) GetAllUsers(ctx context.Context) ([]User, error) {
	return s.repo.ListAll(ctx)
}

func (s *Service) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	login := strings.TrimSpace(req.Login)
	password := strings.TrimSpace(req.Password)
	role := strings.ToUpper(strings.TrimSpace(req.Role))

	if login == "" || password == "" || role == "" {
		return nil, ErrInvalidUserData
	}

	switch role {
	case "CASHIER", "BAKER", "STOREKEEPER", "MANAGER":
	default:
		return nil, errors.New("Недопустимая роль. Доступные роли: CASHIER, BAKER, STOREKEEPER, MANAGER")
	}

	// Проверяем уникальность логина
	existing, err := s.repo.GetByLogin(ctx, login)
	if err == nil && existing != nil {
		return nil, ErrUserAlreadyExists
	}

	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		fullName = login
	}

	newUser := &User{
		FullName:        fullName,
		Login:           login,
		PasswordHash:    password,
		Role:            role,
		IsActive:        true,
		RegisterNumber:  req.RegisterNumber,
		WorkshopStation: req.WorkshopStation,
		WarehouseZoneID: req.WarehouseZoneID,
	}

	// Дефолтные значения для Single Table Inheritance полей
	if role == "CASHIER" && newUser.RegisterNumber == nil {
		reg := 1
		newUser.RegisterNumber = &reg
	} else if role == "BAKER" && (newUser.WorkshopStation == nil || *newUser.WorkshopStation == "") {
		st := "Печной цех №1"
		newUser.WorkshopStation = &st
	} else if role == "STOREKEEPER" && newUser.WarehouseZoneID == nil {
		zone := 101
		newUser.WarehouseZoneID = &zone
	}

	return s.repo.CreateUser(ctx, newUser)
}
