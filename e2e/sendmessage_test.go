package e2e

import (
	"context"
	"crypto/rsa"
	"os"
	"testing"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/markpotocki/messenger/client"
	"github.com/markpotocki/messenger/server"
)

func TestSendAndReceiveMessage(t *testing.T) {
	// set up

	// dummy user data
	userMEPPassword := "HELLO"
	userROOTPassword := "GOODBYE"
	userMEP := server.MakeUser("MEP", userMEPPassword, "mep@example.foo")
	userROOT := server.MakeUser("ROOT", userROOTPassword, "root@example.foo")

	// server
	srvConfig := server.ServerConfig{
		Port: 8080,
	}
	srv := testSetupServer(t, []server.User{userMEP, userROOT})
	srv.Start(context.TODO(), srvConfig)
	serverHost := "http://localhost:8080"

	// client 1
	client1KeyPath := "foo"
	client1 := testSetupClient(t, client1KeyPath, serverHost, userMEP.Username, userMEPPassword)
	// client 2
	client2KeyPath := "bar"
	client2 := testSetupClient(t, client2KeyPath, serverHost, userROOT.Username, userROOTPassword)
	// end set up

	// encrpyt the message
	testMessage := client.MakeClientMessage("ROOT", "MEP", "Hello!")
	rootPubKey := client1.FetchPublicKeyByUserID("ROOT")

	err := client1.SendEncryptedMessage(testMessage, &rootPubKey)
	if err != nil {
		t.Log("failed to send message to server")
		t.Log(err)
		t.FailNow()
	}
	msgs, err := client2.GetMessages("ROOT")
	if err != nil {
		t.Log("error while retrieving ROOT messages")
		t.Log(err)
		t.FailNow()
	}
	if msgs[0].Content != "Hello!" {
		t.Logf("value %s does not match expected %s", msgs[0].Content, "Hello!")
		t.Fail()
	}
	if msgs[0].To != "ROOT" {
		t.Logf("to %s does not match user %s", msgs[0].To, "ROOT")
		t.Fail()
	}
	if msgs[0].From != "MEP" {
		t.Logf("to %s does not match user %s", msgs[0].To, "ROOT")
		t.Fail()
	}

	// clean up
	err = os.Remove(client1KeyPath)
	if err != nil {
		t.Log("clean up unsuccessful for client1")
		t.Log(err)
		t.Fail()
	}
	err = os.Remove(client2KeyPath)
	if err != nil {
		t.Log("clean up unsuccessful for client2")
		t.Log(err)
		t.Fail()
	}

	// debug info if failed
	if t.Failed() {
		jwkKey, err := jwk.New(rootPubKey)
		if err != nil {
			t.Log(err)
		}
		var convertedRSAKey rsa.PublicKey
		if err := jwkKey.Raw(&convertedRSAKey); err != nil {
			t.Log(err)
		}
		t.Logf("JWK Key\n%v", jwkKey)
		t.Logf("RSA Key\n%v", rootPubKey)
		t.Logf("RSA Converted to JWK\n%v", convertedRSAKey)
		t.Logf("Client 2 Key\n%v", client2.PrivateKey.PublicKey)
		t.Logf("Test Message Content\n%v", testMessage.Content)
		t.Logf("Received Message Content\n%v", msgs[0].Content)
	}
}

// setupServer sets up a server and adds two dummy users to it for testing
func testSetupServer(t *testing.T, users []server.User) *server.Server {
	userStore := server.MakeMemoryUserStore()

	for _, user := range users {
		err := userStore.Add(user)
		if err != nil {
			t.Log("failed to add user")
			t.Log(err)
			t.FailNow()
		}
	}

	srv := server.Server{
		Keystore:     server.MakeMemoryUserKeystore(),
		MessageStore: server.MakeMemoryMessageStore(),
		UserStore:    userStore,
	}
	return &srv
}

func testSetupClient(t *testing.T, keyPath string, host string, username, password string) *client.Client {
	client1 := client.MakeClient(keyPath, host)
	client1.SetBasicAuth(username, password)
	err := client1.RegisterKey("MEP")
	if err != nil {
		t.Log("failed to register client1 key")
		t.Log(err)
		t.FailNow()
	}
	return client1
}
