package config

import (
	"fmt"
	"github.com/google/uuid"
	"sync"
)

// User представляет пользователя в системе (аналог VLESS)
type User struct {
	ID    uuid.UUID `json:"id"`              // UUID пользователя
	Email string    `json:"email,omitempty"` // Email для идентификации
	Level int       `json:"level,omitempty"` // Уровень пользователя (0 = default)
	Flow  string    `json:"flow,omitempty"`  // Flow type (например, "xtls-rprx-vision")
}

// UserValidator управляет пользователями и выполняет валидацию
type UserValidator struct {
	users map[uuid.UUID]*User
	mu    sync.RWMutex
}

// NewUserValidator создает новый валидатор пользователей
func NewUserValidator(users []User) *UserValidator {
	validator := &UserValidator{
		users: make(map[uuid.UUID]*User, len(users)),
	}

	for i := range users {
		validator.users[users[i].ID] = &users[i]
	}

	return validator
}

// Validate проверяет, существует ли пользователь с данным UUID
func (v *UserValidator) Validate(userID uuid.UUID) (*User, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	user, exists := v.users[userID]
	return user, exists
}

// AddUser добавляет нового пользователя
func (v *UserValidator) AddUser(user User) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, exists := v.users[user.ID]; exists {
		return fmt.Errorf("user with ID %s already exists", user.ID)
	}

	v.users[user.ID] = &user
	return nil
}

// RemoveUser удаляет пользователя
func (v *UserValidator) RemoveUser(userID uuid.UUID) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, exists := v.users[userID]; !exists {
		return fmt.Errorf("user with ID %s not found", userID)
	}

	delete(v.users, userID)
	return nil
}

// GetUser возвращает пользователя по UUID
func (v *UserValidator) GetUser(userID uuid.UUID) (*User, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	user, exists := v.users[userID]
	if !exists {
		return nil, fmt.Errorf("user with ID %s not found", userID)
	}

	return user, nil
}

// ListUsers возвращает список всех пользователей
func (v *UserValidator) ListUsers() []User {
	v.mu.RLock()
	defer v.mu.RUnlock()

	users := make([]User, 0, len(v.users))
	for _, user := range v.users {
		users = append(users, *user)
	}

	return users
}

// Count возвращает количество пользователей
func (v *UserValidator) Count() int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return len(v.users)
}
