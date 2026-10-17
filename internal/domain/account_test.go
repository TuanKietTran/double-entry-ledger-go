package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestAccountValidate(t *testing.T) {
	valid := Account{ID: uuid.New(), Currency: "USD", Type: Asset, Status: AccountActive}
	for name, account := range map[string]Account{
		"valid":            valid,
		"missing id":       {Currency: "USD", Type: Asset, Status: AccountActive},
		"invalid currency": {ID: uuid.New(), Currency: "usd", Type: Asset, Status: AccountActive},
		"invalid type":     {ID: uuid.New(), Currency: "USD", Type: "UNKNOWN", Status: AccountActive},
		"invalid status":   {ID: uuid.New(), Currency: "USD", Type: Asset, Status: "PAUSED"},
	} {
		t.Run(name, func(t *testing.T) {
			gotErr := account.Validate()
			if name == "valid" && gotErr != nil {
				t.Fatalf("valid account rejected: %v", gotErr)
			}
			if name != "valid" && gotErr == nil {
				t.Fatal("invalid account accepted")
			}
		})
	}
}
