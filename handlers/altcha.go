package handlers

import (
	"net/http"

	"github.com/altcha-org/altcha-lib-go"
	"github.com/pocketbase/pocketbase/core"
)

func AltchaChallenge(hmacKey string) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {

		challenge, err := altcha.CreateChallenge(altcha.ChallengeOptions{
			HMACKey:   hmacKey,
			MaxNumber: 100000,
		})

		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return e.JSON(http.StatusOK, challenge)
	}
}
