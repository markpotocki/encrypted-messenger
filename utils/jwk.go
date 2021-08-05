package utils

import (
	"crypto/rsa"
	"errors"

	"github.com/lestrrat-go/jwx/jwk"
)

type JWKSet struct {
	KeyType        string `json:"kty"`
	Algorithum     string `json:"alg"`
	Curve          string `json:"crv"`
	CoordinateX    string `json:"x"`
	CoordinateY    string `json:"y"`
	Use            string `json:"use"`
	PublicModulus  string `json:"n"`
	PublicExponent string `json:"e"`
	KeyID          string `json:"kid"`
}

func MakeJWKSetFromRSAPublicKey(publicKey *rsa.PublicKey) (jwk.RSAPublicKey, error) {
	key, err := jwk.New(publicKey)
	if err != nil {
		return nil, err
	}
	if jwkRSAKey, ok := key.(jwk.RSAPublicKey); ok {
		return jwkRSAKey, nil
	}
	return nil, errors.New("rsa public key could not be cast to jwk public key")
}

func MakePublicKeyFromJWK(key jwk.Key) (*rsa.PublicKey, error) {
	var rsaKey rsa.PublicKey
	if err := key.Raw(&rsaKey); err != nil {
		return nil, err
	}
	return &rsaKey, nil
}
