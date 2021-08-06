package server

import (
	"fmt"
	"sync"
	"testing"
)

func TestMakeMemoryMessageStore(t *testing.T) {
	expectedMessageLength := 0
	messageStore := MakeMemoryMessageStore()
	actualMessageLength := len(messageStore.messages)
	if !assert(expectedMessageLength, actualMessageLength) {
		t.Log(sprintFailure(expectedMessageLength, actualMessageLength))
	}
}

func TestMemoryMessageStoreAdd(t *testing.T) {
	tests := []struct {
		name                  string
		expectedMessageLength int
		expectedMessages      []Message
		expectedError         error
	}{
		// test message length 1
		{
			name:                  "TestAddLength-1",
			expectedMessageLength: 1,
			expectedMessages: []Message{
				{To: "MEP", From: "PEM", Content: "Test"},
			},
			expectedError: nil,
		},
		// test message length 2
		{
			name:                  "TestAddLength-2",
			expectedMessageLength: 2,
			expectedMessages: []Message{
				{ID: "1", To: "MEP", From: "PEM", Content: "Hello"},
				{ID: "2", To: "MEP", From: "BLAH", Content: "Goodbye"},
			},
			expectedError: nil,
		},
		// test duplicate add
		{
			name:                  "TestAddDuplicate",
			expectedMessageLength: 1,
			expectedMessages: []Message{
				{ID: "1", To: "MEP", From: "PEM", Content: "Hello"},
				{ID: "1", To: "MEP", From: "PEM", Content: "Hello"},
			},
			expectedError: ErrDuplicateID{ID: "1", Action: "Add"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			messageStore := MemoryMessageStore{
				messages: make(map[MessageID]Message),
				mutex:    &sync.Mutex{},
			}

			var err error
			for _, msg := range test.expectedMessages {
				err = messageStore.Add(msg)
			}

			// test if we don't have the expected error
			if err != test.expectedError {
				t.Log(err)
				t.Fail()
			}

			// verify message length
			actualMessageLength := len(messageStore.messages)
			if !assert(test.expectedMessageLength, actualMessageLength) {
				t.Log(sprintFailure(test.expectedMessageLength, actualMessageLength))
				t.Fail()
			}

			// verify messages equal
			for _, val := range test.expectedMessages {
				if !assert(val, messageStore.messages[MessageID(val.ID)]) {
					t.Log(sprintFailure(val, messageStore.messages[MessageID(val.ID)]))
					t.Fail()
				}
			}
		})
	}

}

func TestMemoryMessageStoreFindReceivedByUserID(t *testing.T) {
	to := "MEP"
	messages := []Message{
		{ID: "0", To: to, From: "Who"},
		{ID: "1", To: to, From: "Where"},
		{ID: "2", To: "No", From: "What"},
	}

	expectedMessages := []Message{
		{ID: "0", To: to, From: "Who"},
		{ID: "1", To: to, From: "Where"},
	}
	expectedLength := len(expectedMessages)

	messageStore := MemoryMessageStore{
		messages: map[MessageID]Message{},
		mutex:    &sync.Mutex{},
	}

	for _, message := range messages {
		messageStore.messages[MessageID(message.ID)] = message
	}

	actualMessages, err := messageStore.FindReceivedByUserID(to)
	if err != nil {
		t.Error(err)
	}

	// check length
	if !assert(expectedLength, len(actualMessages)) {
		t.Error(sprintFailure(expectedLength, len(actualMessages)))
	}

	// check we only got our messages back
	for index, msg := range actualMessages {
		if !assert(expectedMessages[index], msg) {
			t.Error(sprintFailure(expectedMessages[index], msg))
		}
	}

}

func TestMemoryMessageStoreFindSentByUserID(t *testing.T) {
	from := "MEP"
	messages := []Message{
		{ID: "0", To: "Who", From: from},
		{ID: "1", To: "Where", From: from},
		{ID: "2", To: "What", From: "What"},
	}

	expectedMessages := []Message{
		{ID: "0", To: "Who", From: from},
		{ID: "1", To: "Where", From: from},
	}
	expectedLength := len(expectedMessages)

	messageStore := MemoryMessageStore{
		messages: map[MessageID]Message{},
		mutex:    &sync.Mutex{},
	}

	for _, message := range messages {
		messageStore.messages[MessageID(message.ID)] = message
	}

	actualMessages, err := messageStore.FindSentByUserID(from)
	if err != nil {
		t.Error(err)
	}

	// check length
	if !assert(expectedLength, len(actualMessages)) {
		t.Error(sprintFailure(expectedLength, len(actualMessages)))
	}

	// check we only got our messages back
	for index, msg := range actualMessages {
		if !assert(expectedMessages[index], msg) {
			t.Error(sprintFailure(expectedMessages[index], msg))
		}
	}
}

func TestMemoryMessageStoreFindAllByUserID(t *testing.T) {
	to := "MEP"
	expectedMessages := []Message{
		{ID: "0", To: to, From: "Who"},
		{ID: "1", To: to, From: "Where"},
		{ID: "2", To: "No", From: to},
	}
	expectedLength := len(expectedMessages)

	messageStore := MemoryMessageStore{
		messages: map[MessageID]Message{},
		mutex:    &sync.Mutex{},
	}

	for _, message := range expectedMessages {
		messageStore.messages[MessageID(message.ID)] = message
	}

	actualMessages, err := messageStore.FindAllByUserID(to)
	if err != nil {
		t.Error(err)
	}

	// check length
	if !assert(expectedLength, len(actualMessages)) {
		t.Error(sprintFailure(expectedLength, len(actualMessages)))
	}

	// check we only got our messages back
	for _, msg := range actualMessages {
		var found bool
		for _, msg2 := range expectedMessages {
			if msg == msg2 {
				found = true
				break
			}
		}
		if !found {
			t.Error("message", msg, "not found in", expectedMessages)
		}
		found = false
	}
}

func TestMemoryMessageStoreDeleteByID(t *testing.T) {
	tests := []struct {
		name            string
		addMessage      Message
		deleteMessageID string
		expectedError   error
	}{
		{"DeleteByIDOK", Message{ID: "1"}, "1", nil},
		{"DeleteByIDDoesNotExist", Message{ID: "1"}, "foo", ErrKeyDoesNotExist{"foo"}},
	}

	for _, test := range tests {
		messageStore := MemoryMessageStore{
			messages: make(map[MessageID]Message),
			mutex:    &sync.Mutex{},
		}
		messageStore.messages[MessageID(test.addMessage.ID)] = test.addMessage
		err := messageStore.DeleteByID(MessageID(test.deleteMessageID))
		if err != test.expectedError {
			t.Error(sprintFailure(test.expectedError, err))
		}
	}

}

func assert(expected interface{}, actual interface{}) bool {
	return expected == actual
}

func sprintFailure(expected interface{}, actual interface{}) string {
	return fmt.Sprintf("assertion failed expected [%v] actual [%v]", expected, actual)
}
