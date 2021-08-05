package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/markpotocki/messenger/client"
	"github.com/markpotocki/messenger/server"
	"github.com/markpotocki/messenger/utils"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			startServer()
		}
	}()
	if prog := os.Args[1]; prog != "" {
		switch prog {
		case "server":
			startServer()
		case "client":
			startClient()
		}
	} else {
		fmt.Println("use either server or client as args")
	}
}

func startServer() {
	// the server
	srv := server.Server{
		Keystore:     server.MakeMemoryUserKeystore(),
		MessageStore: server.MakeMemoryMessageStore(),
	}
	serverConfig := server.ServerConfig{
		Address: "",
		Port:    8080,
		TLS:     false,
	}
	rootContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	errchan := srv.Start(rootContext, serverConfig)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-quit:
		utils.LogInfo("shutting down http server")
	case err := <-errchan:
		utils.LogError(err.Error())
	}
}

func startClient() {
	flagSendMessages := flag.Bool("send", false, "set flag to send a message")
	flagMessageTo := flag.String("to", "", "set when sending messages as to field")
	flagMessageFrom := flag.String("from", "", "set when sending message as from field")
	flagMessageContent := flag.String("content", "", "set when sending message as content field")
	flagUsername := flag.String("username", "", "username to use for sending messages")
	flag.Parse()
	// start the client
	// the client #1
	fmt.Println(*flagUsername)
	cli := client.MakeClient("priv_key.gogob", "http://localhost:8080")
	err := cli.RegisterKey(*flagUsername)
	if err != nil {
		log.Println(err)
	}

	if *flagSendMessages {
		pubKey := cli.FetchPublicKeyByUserID(*flagMessageTo)
		message := client.MakeClientMessage(*flagMessageTo, *flagMessageFrom, *flagMessageContent)
		message, err = message.EncryptContent(&pubKey)
		if err != nil {
			panic(err)
		}
		cli.SendMessage(message)
	} else {
		messages, err := cli.GetMessages(*flagUsername)
		if err != nil {
			panic(err)
		}
		fmt.Println(messages)
	}
}
