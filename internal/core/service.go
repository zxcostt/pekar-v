package core

import (
	"context"
	"errors"
)

var (
	ErrInvalidCredentials = errors.New("Неверный логин или пароль. Повторите попытку")
	ErrAccountBlocked     = errors.New("Учетная запись заблокирована. Обратитесь к управляющему пекарни")
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
