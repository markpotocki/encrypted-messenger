package client

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"encoding/base64"

	"github.com/markpotocki/messenger/types"
)

type ClientMessage types.Message

func (message ClientMessage) EncryptContent(toPublicKey *rsa.PublicKey) (ClientMessage, error) {
	encrpytedContent, err := rsa.EncryptPKCS1v15(cryptorand.Reader, toPublicKey, []byte(message.Content))
	if err != nil {
		return message, err
	}
	message.Content = base64.URLEncoding.EncodeToString(encrpytedContent)
	message.Encrypted = true
	return message, nil
}

func (message ClientMessage) DecryptContent(myPrivateKey *rsa.PrivateKey) (ClientMessage, error) {
	unencoded, err := base64.URLEncoding.DecodeString(message.Content)
	if err != nil {
		return message, err
	}
	decryptedContent, err := rsa.DecryptPKCS1v15(cryptorand.Reader, myPrivateKey, unencoded)
	if err != nil {
		return message, err
	}
	message.Content = string(decryptedContent)
	message.Encrypted = false
	return message, nil
}
