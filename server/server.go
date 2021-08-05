package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/markpotocki/messenger/types"
	"github.com/markpotocki/messenger/utils"
	"golang.org/x/net/websocket"
)

type Server struct {
	UserStore    UserStore
	Keystore     UserKeystore
	MessageStore MessageStore
}

type ServerConfig struct {
	Address string
	Port    int
	TLS     bool
}

func (server *Server) AddUser(w http.ResponseWriter, r *http.Request) {
	// check if it is POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// register the user
	keyData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err) // TODO handle it
	}
	key, err := jwk.ParseKey(keyData)
	if err != nil {
		panic(err) // TODO handle it
	}

	jwkKey := key.(jwk.RSAPublicKey)

	// get the user from context
	user := GetUserFromContext(r.Context())

	registerRequest := types.UserRegisterRequest{
		UserID:    user.Username,
		PublicKey: jwkKey,
	}

	if err := server.Keystore.AddPublicKey(registerRequest.UserID, registerRequest.PublicKey); err != nil {
		utils.LogError("failed to register new user")
		utils.LogError(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	utils.LogDebug("added new user")
	w.WriteHeader(http.StatusCreated)
}

func (server *Server) GetPublicKeyByUser(w http.ResponseWriter, r *http.Request) {
	// find the query param for userID
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		// cannot be blank
		utils.LogDebug("blank userIDs cannot be used")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// find the user
	pubKey, err := server.Keystore.PublicKeyByUserID(userID)
	if err != nil {
		utils.LogError("could not retrieve users public key")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// return the pub key
	jwkKey, err := utils.MakeJWKSetFromRSAPublicKey(&pubKey)
	if err != nil {
		utils.LogError("error converting JWK to RSAPublicKey")
		utils.LogError(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(jwkKey)
	if err != nil {
		utils.LogError("failed to encode JWK to json")
		utils.LogError(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	buffer := bytes.NewBuffer(data)
	io.Copy(w, buffer)
}

func (server *Server) Start(ctx context.Context, config ServerConfig) chan error {
	// register handlers
	var keyHandler http.Handler = mutliMethodHandler{
		handlers: map[string]http.HandlerFunc{
			"GET":  server.GetPublicKeyByUser,
			"POST": server.AddUser,
		},
	}
	var messageHandler http.Handler = mutliMethodHandler{
		handlers: map[string]http.HandlerFunc{
			"GET":  server.GetMessages,
			"POST": server.AddMessage,
		},
	}

	keyHandler = coorsHandler{
		next: keyHandler,
	}

	messageHandler = coorsHandler{
		next: messageHandler,
	}

	http.HandleFunc("/pubkey", server.AuthenticateMiddleware(keyHandler))
	http.HandleFunc("/messages", server.AuthenticateMiddleware(messageHandler))
	utils.LogInfo(fmt.Sprintf("starting http server on %s:%d", config.Address, config.Port))
	errChan := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
			utils.LogInfo("shutting down http server")
			return
		default:
			errChan <- http.ListenAndServe(fmt.Sprintf("%s:%d", config.Address, config.Port), nil)
		}
	}()
	return errChan
}

func (server *Server) AddMessage(w http.ResponseWriter, r *http.Request) {
	// must be post
	if r.Method != http.MethodPost {
		utils.LogDebug(fmt.Sprintf("server.AddMessage %s is not allowed must be POST", r.Method))
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// decode
	var message Message
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&message); err != nil {
		utils.LogDebug("server.AddMessage failed to decode request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// add
	if err := server.MessageStore.Add(message); err != nil {
		utils.LogDebug("unable to add message to store")
		utils.LogError(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (server *Server) GetMessages(w http.ResponseWriter, r *http.Request) {
	// must be GET
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// ID is query as userID
	id := r.URL.Query().Get("userID")
	messages, err := server.MessageStore.FindAllByUserID(id)
	if err != nil {
		utils.LogError(fmt.Sprintf("server.GetMessages %s", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(messages); err != nil {
		utils.LogError(fmt.Sprintf("server.GetMessages %s", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (server *Server) WebSocketMessageHandler(ws *websocket.Conn) {
	// read inbound messages and save them
	go func() {
		var message Message
		decoder := json.NewDecoder(ws)
		if err := decoder.Decode(&message); err != nil {
			utils.LogError(err.Error())
			ws.WriteClose(500)
		}
		if err := server.MessageStore.Add(message); err != nil {
			utils.LogError(err.Error())
			ws.WriteClose(500)
		}
	}()

	// outbound messages
	go func() {
		// TODO
	}()
}

type mutliMethodHandler struct {
	handlers map[string]http.HandlerFunc
}

func (handler mutliMethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := handler.handlers[r.Method]; ok {
		h.ServeHTTP(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

type coorsHandler struct {
	next http.Handler
}

func (handler coorsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Add("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	handler.next.ServeHTTP(w, r)
}

func (server *Server) AuthenticateMiddleware(next http.Handler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// authenticate user now
		user, pwd, ok := r.BasicAuth()
		if ok {
			u, err := server.UserStore.Find(user)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			// user exists, validate entry
			isAuthed := u.Authenticate(pwd)
			if isAuthed {
				r = r.WithContext(AddUserToContext(r.Context(), u))
				next.ServeHTTP(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
	}
}
