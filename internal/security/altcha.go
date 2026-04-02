package security

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
)

func generateKey() string {
	b := make([]byte, 32)

	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}

func GetAltchaKey() string {
	key := os.Getenv("ALTCHA_HMAC_KEY")

	if key != "" {
		log.Println("ALTCHA using configured HMAC key")
		return key
	}

	log.Println("ALTCHA generating temporary HMAC key")
	return generateKey()
}
