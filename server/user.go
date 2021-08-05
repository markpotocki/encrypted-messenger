package server

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type ContextKey string

const (
	principalKey ContextKey = "PRINCIPAL"
)

type UserStore interface {
	Add(user User) error
	Delete(user User) error
	Find(userID string) (User, error)
}

type MemoryUserStore struct {
	users map[string]User
	mutex *sync.Mutex
}

func MakeMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		users: make(map[string]User),
		mutex: &sync.Mutex{},
	}
}

func (us *MemoryUserStore) Add(user User) error {
	// check if exists
	if _, ok := us.users[user.Username]; ok {
		return ErrUserAlreadyExists{
			Username: user.Username,
		}
	}
	// add
	us.mutex.Lock()
	defer us.mutex.Unlock()
	us.users[user.Username] = user
	return nil
}

func (us *MemoryUserStore) Delete(user User) error {
	// check if exists
	if _, ok := us.users[user.Username]; ok {
		return ErrUserDoesNotExist{
			Username: user.Username,
		}
	}
	// delete
	us.mutex.Lock()
	defer us.mutex.Unlock()
	delete(us.users, user.Username)
	return nil
}

func (us *MemoryUserStore) Find(username string) (User, error) {
	if u, ok := us.users[username]; ok {
		return u, nil
	}
	return User{}, ErrUserDoesNotExist{
		Username: username,
	}
}

type User struct {
	Username string
	Password []byte
	Email    string
}

func MakeUser(username, password, email string) User {
	encrpytedPassword := encodePassword(password)
	return User{
		Username: username,
		Password: encrpytedPassword,
		Email:    email,
	}
}

func (user User) Authenticate(password string) bool {
	return validatePassword(password, user.Password)
}

func encodePassword(password string) []byte {
	encrytedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return encrytedPassword
}

func validatePassword(plaintext string, encryptedText []byte) bool {
	err := bcrypt.CompareHashAndPassword(encryptedText, []byte(plaintext))
	return err == nil
}

func AddUserToContext(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, principalKey, user)
}

func GetUserFromContext(ctx context.Context) User {
	return ctx.Value(principalKey).(User)
}

type ErrUserAlreadyExists struct {
	Username string
}

func (err ErrUserAlreadyExists) Error() string {
	return fmt.Sprintf("user %s already exists", err.Username)
}

type ErrUserDoesNotExist struct {
	Username string
}

func (err ErrUserDoesNotExist) Error() string {
	return fmt.Sprintf("user %s does not exist", err.Username)
}
