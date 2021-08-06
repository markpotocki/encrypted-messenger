package server

import (
	"fmt"
	"sync"

	"github.com/markpotocki/messenger/types"
)

type MessageID types.MessageID
type Message types.Message

type MessageStore interface {
	Add(message Message) error
	DeleteByID(messageID MessageID) error
	FindReceivedByUserID(userID string) ([]Message, error)
	FindSentByUserID(userID string) ([]Message, error)
	FindAllByUserID(userID string) ([]Message, error)
}

type MemoryMessageStore struct {
	messages map[MessageID]Message
	mutex    *sync.Mutex
}

func MakeMemoryMessageStore() *MemoryMessageStore {
	return &MemoryMessageStore{
		messages: make(map[MessageID]Message),
		mutex:    &sync.Mutex{},
	}
}

func (store *MemoryMessageStore) Add(message Message) error {
	if _, ok := store.messages[MessageID(message.ID)]; ok {
		return ErrDuplicateID{
			ID:     message.ID,
			Action: "Add",
		}
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.messages[MessageID(message.ID)] = message
	return nil
}

func (store *MemoryMessageStore) DeleteByID(messageID MessageID) error {
	if _, ok := store.messages[messageID]; !ok {
		return ErrKeyDoesNotExist{
			key: string(messageID),
		}
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	delete(store.messages, messageID)
	return nil
}

func (store *MemoryMessageStore) FindReceivedByUserID(userID string) ([]Message, error) {
	messages := make([]Message, 0)
	for _, message := range store.messages {
		if message.To == userID {
			messages = append(messages, message)
		}
	}
	return messages, nil
}

func (store *MemoryMessageStore) FindSentByUserID(userID string) ([]Message, error) {
	messages := make([]Message, 0)
	for _, message := range store.messages {
		if message.From == userID {
			messages = append(messages, message)
		}
	}
	return messages, nil
}

func (store *MemoryMessageStore) FindAllByUserID(userID string) ([]Message, error) {
	messages := make([]Message, 0)
	for _, message := range store.messages {
		if message.From == userID || message.To == userID {
			messages = append(messages, message)
		}
	}
	return messages, nil
}

type ErrDuplicateID struct {
	ID     types.MessageID
	Action string
}

func (err ErrDuplicateID) Error() string {
	return fmt.Sprintf("duplication id of %s on action %s", err.ID, err.Action)
}
