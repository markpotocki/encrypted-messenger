package types

import (
	"encoding/base64"
	"math/rand"
	"time"
)

const (
	MessageIDSize = 64
)

type Message struct {
	From      string
	To        string
	TimeSent  time.Time
	ID        MessageID
	Content   string
	Encrypted bool
}

func MakeMessage(from string, to string, content string) Message {
	return Message{
		From:      from,
		To:        to,
		TimeSent:  time.Now(), // this makes an assumption that messaging creation means message sending
		ID:        MakeMessageID(),
		Content:   content,
		Encrypted: false,
	}
}

type MessageID string

func MakeMessageID() MessageID {
	id := make([]byte, MessageIDSize)
	if _, err := rand.Read(id); err != nil {
		panic(err)
	}
	return MessageID(base64.URLEncoding.EncodeToString(id))
}

func init() {
	rand.Seed(time.Now().Unix())
}
