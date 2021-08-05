package server

import (
	"crypto/rsa"
	"fmt"
	"sync"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/markpotocki/messenger/utils"
)

type UserKeystore interface {
	PublicKeyByUserID(userID string) (rsa.PublicKey, error)
	AddPublicKey(userID string, publicKey jwk.RSAPublicKey) error
	DeletePublicKeyByUserID(userID string) error
}

type MemoryUserKeystore struct {
	keys  map[string]rsa.PublicKey
	mutex *sync.Mutex
}

func MakeMemoryUserKeystore() *MemoryUserKeystore {
	return &MemoryUserKeystore{
		keys:  make(map[string]rsa.PublicKey),
		mutex: &sync.Mutex{},
	}
}

func (keystore MemoryUserKeystore) PublicKeyByUserID(userID string) (rsa.PublicKey, error) {
	if key, ok := keystore.keys[userID]; ok {
		return key, nil
	}
	// no key found
	return rsa.PublicKey{}, ErrKeyDoesNotExist{key: userID}
}

func (keystore MemoryUserKeystore) AddPublicKey(userID string, publicKey jwk.RSAPublicKey) error {
	if _, ok := keystore.keys[userID]; ok {
		return ErrKeyAlreadyExists{key: userID}
	}
	// decode to rsa key
	rsaKey, err := utils.MakePublicKeyFromJWK(publicKey)
	if err != nil {
		return err
	}

	keystore.mutex.Lock()
	defer keystore.mutex.Unlock()
	keystore.keys[userID] = *rsaKey
	return nil
}

func (keystore MemoryUserKeystore) DeletePublicKeyByUserID(userID string) error {
	if _, ok := keystore.keys[userID]; ok {
		keystore.mutex.Lock()
		defer keystore.mutex.Unlock()
		delete(keystore.keys, userID)
		return nil
	}
	return ErrKeyDoesNotExist{key: userID}
}

type ErrKeyDoesNotExist struct {
	key string
}

func (err ErrKeyDoesNotExist) Error() string {
	return fmt.Sprintf("key %s does not exist", err.key)
}

type ErrKeyAlreadyExists struct {
	key string
}

func (err ErrKeyAlreadyExists) Error() string {
	return fmt.Sprintf("key %s already exists", err.key)
}
