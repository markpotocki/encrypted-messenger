package types

import (
	"github.com/lestrrat-go/jwx/jwk"
)

type UserRegisterRequest struct {
	UserID    string
	PublicKey jwk.RSAPublicKey
}
