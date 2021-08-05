package client

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/markpotocki/messenger/types"
	"github.com/markpotocki/messenger/utils"
)

const (
	sizeKey = 2048
	pathKey = "priv_key.gogob"
)

func MakeClientMessage(to string, from string, content string) ClientMessage {
	return ClientMessage(types.MakeMessage(from, to, content))
}

type principal struct {
	Username string
	Password string
}

type Client struct {
	PrivateKey *rsa.PrivateKey
	ServerHost string
	Principal  principal
	// client http.Client
}

func MakeClient(keyPath string, serverHost string) *Client {
	key, err := loadKey(keyPath)
	if err != nil {
		utils.LogWarn("generating new key pair for client")
		key, err = generateAndSaveKey(keyPath)
		if err != nil {
			panic(err)
		}
	}

	return &Client{
		PrivateKey: key,
		ServerHost: serverHost,
	}
}

func (cli *Client) SetBasicAuth(username string, password string) {
	cli.Principal.Username = username
	cli.Principal.Password = password
}

func (cli *Client) RegisterKey(userID string) error {
	jwkKey, err := utils.MakeJWKSetFromRSAPublicKey(&cli.PrivateKey.PublicKey)
	if err != nil {
		return err
	}

	keyData, err := json.Marshal(jwkKey)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(keyData)

	request, err := http.NewRequest(http.MethodPost, cli.ServerHost+"/pubkey", buffer)
	if err != nil {
		return err
	}
	request.SetBasicAuth(cli.Principal.Username, cli.Principal.Password)

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		utils.LogError(fmt.Sprint("status of", resp.Status))
		return errors.New("bad status")
	}
	defer resp.Body.Close()
	return nil
}

func (cli *Client) FetchPublicKeyByUserID(userID string) rsa.PublicKey {
	// build request
	request, err := http.NewRequest(http.MethodGet, cli.ServerHost+"/pubkey", nil)
	if err != nil {
		panic(err)
	}
	request.SetBasicAuth(cli.Principal.Username, cli.Principal.Password)
	query := request.URL.Query()
	query.Add("userID", userID)
	request.URL.RawQuery = query.Encode()

	// make request
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusOK {
		panic("bad status of " + resp.Status)
	}

	jwkSet, err := jwk.ParseReader(resp.Body)
	if err != nil {
		panic(err)
	}

	jwkKey, ok := jwkSet.Get(0)
	if !ok {
		panic("there is no jwkKeys in provided set")
	}

	pubKey, err := utils.MakePublicKeyFromJWK(jwkKey)
	if err != nil {
		panic(err)
	}
	return *pubKey
}

func (cli *Client) SendMessage(message ClientMessage) error {
	marshMessage, err := json.Marshal(message)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(marshMessage)

	request, err := http.NewRequest(http.MethodPost, cli.ServerHost+"/messages", buffer)
	request.SetBasicAuth(cli.Principal.Username, cli.Principal.Password)
	if err != nil {
		return err
	}
	utils.LogInfo("sending message")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprint("bad status of", resp.Status))
	}
	return nil
}

func (cli *Client) SendEncryptedMessage(message ClientMessage, key *rsa.PublicKey) error {
	msg, err := message.EncryptContent(key)
	if err != nil {
		return err
	}
	msg.Encrypted = true
	if err := cli.SendMessage(msg); err != nil {
		return err
	}
	return nil
}

func (cli *Client) GetMessages(userID string) ([]ClientMessage, error) {
	request, err := http.NewRequest(http.MethodGet, cli.ServerHost+"/messages", nil)
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(cli.Principal.Username, cli.Principal.Password)
	query := request.URL.Query()
	query.Add("userID", userID)
	request.URL.RawQuery = query.Encode()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("client.GetMessages status of " + response.Status)
	}

	// decode json
	var messages []ClientMessage
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&messages); err != nil {
		return nil, err
	}

	// decrypt
	for i, message := range messages {
		m, _ := message.DecryptContent(cli.PrivateKey)
		messages[i] = m
	}

	return messages, nil
}

func loadKey(keyPath string) (*rsa.PrivateKey, error) {
	// load the file containing our private key
	keyFile, err := os.Open(keyPath)
	if err != nil {
		utils.LogError("unable to open private key file")
		return nil, err
	}
	defer keyFile.Close()
	data, err := ioutil.ReadAll(keyFile)
	if err != nil {
		return nil, err
	}
	for len(data) != 0 {
		block, rest := pem.Decode(data)
		if block == nil {
			utils.LogWarn("generating new keypair")
			return generateAndSaveKey(keyPath)
		}
		switch block.Type {
		case "RSA PRIVATE KEY":
			pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			return pk, nil
		}
		data = rest
	}
	return nil, errors.New("unexpected end")
}

func generateAndSaveKey(keyPath string) (*rsa.PrivateKey, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, sizeKey)
	if err != nil {
		utils.LogError("unable to generate private key")
		return nil, err
	}
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privKey)
	publicKeyBytes := x509.MarshalPKCS1PublicKey(&privKey.PublicKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	// file saving
	fileKey, err := os.Create(keyPath)
	if err != nil {
		utils.LogError("unable to create private key file")
		return nil, err
	}
	defer fileKey.Close()
	err = pem.Encode(fileKey, privateKeyBlock)
	if err != nil {
		utils.LogError("failed to encode private key")
		return nil, err
	}
	err = pem.Encode(fileKey, publicKeyBlock)
	if err != nil {
		utils.LogError("failed to encode public key")
		return nil, err
	}

	return privKey, nil
}
