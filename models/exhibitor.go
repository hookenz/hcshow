package models

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/pocketbase/pocketbase/tools/types"
)

type ExhibitorRow struct {
	Id          string
	ExhibitorID string
	FirstName   string
	LastName    string
	BirthDate   string
	Age         int
}

func BirthDateForInput(dt types.DateTime) string {
	t := dt.Time()
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func BuildExhibitorRow(id, exhibitorID, firstName, lastName string, birthDT types.DateTime) ExhibitorRow {
	now := time.Now()
	t := birthDT.Time()
	birthDate := "—"
	age := 0
	if !t.IsZero() {
		birthDate = t.Format("2 Jan 2006")
		age = now.Year() - t.Year()
		if now.Month() < t.Month() || (now.Month() == t.Month() && now.Day() < t.Day()) {
			age--
		}
	}
	return ExhibitorRow{
		Id:          id,
		ExhibitorID: exhibitorID,
		FirstName:   firstName,
		LastName:    lastName,
		BirthDate:   birthDate,
		Age:         age,
	}
}

func GenerateExhibitorID() (string, error) {
	year := time.Now().Year() % 100

	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", err
	}
	digits := fmt.Sprintf("%04d", n.Int64())

	sum := year
	for _, c := range digits {
		sum += int(c - '0')
	}
	check := sum % 10

	return fmt.Sprintf("%02d%s%d", year, digits, check), nil
}
