package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/golang-jwt/jwe"
)

func main() {
	// Generate a 2048-bit RSA key pair
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Error generating key pair:", err)
		return
	}

	// Get the public key from the private key
	pub := priv.Public().(*rsa.PublicKey) // cast to *rsa.PublicKey type

	// Encode the private key to PEM format
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})
	fmt.Println("Private key in PEM format:", string(privPEM))

	// Encode the public key to PEM format
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		fmt.Println("Error encoding public key:", err)
		return
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	fmt.Println("Public key in PEM format:", string(pubPEM))

	// Generate a JWE token with the public key and the payload
	// The payload can be any byte slice, such as a JWT or a plain text
	payload := []byte("Hello, world!")
	token, err := jwe.NewJWE(jwe.KeyAlgorithmRSAOAEP, pub, jwe.EncryptionTypeA256GCM, payload)
	if err != nil {
		fmt.Println("Error creating JWE:", err)
		return
	}

	// Serialize the token to a compact string format
	compact, err := token.CompactSerialize()
	if err != nil {
		fmt.Println("Error serializing JWE:", err)
		return
	}

	fmt.Println("Generated JWE:", compact)

	// Parse and decrypt the JWE token with the private key
	parsedToken, err := jwe.ParseEncrypted(compact)
	if err != nil {
		fmt.Println("Error parsing JWE:", err)
		return
	}
	// Decrypt the JWE token with the private key
	decrypted, err := parsedToken.Decrypt(priv) // priv is a *rsa.PrivateKey
	if err != nil {
		fmt.Println("Error decrypting JWE:", err)
		return
	}

	// Convert the decrypted payload to a string
	decryptedString := string(decrypted)
	fmt.Println("Decrypted payload:", decryptedString)

}
