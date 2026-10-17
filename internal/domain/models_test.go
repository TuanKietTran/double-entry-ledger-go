package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestTransactionIsBalanced(t *testing.T) {
	tests := []struct {
		name    string
		entries []Entry
		want    bool
	}{
		{
			name: "equal debit and credit",
			entries: []Entry{
				{Amount: decimal.NewFromInt(100), Direction: Debit},
				{Amount: decimal.NewFromInt(100), Direction: Credit},
			},
			want: true,
		},
		{
			name: "split credits match debit",
			entries: []Entry{
				{Amount: decimal.RequireFromString("10.50"), Direction: Debit},
				{Amount: decimal.RequireFromString("4.25"), Direction: Credit},
				{Amount: decimal.RequireFromString("6.25"), Direction: Credit},
			},
			want: true,
		},
		{
			name: "unbalanced",
			entries: []Entry{
				{Amount: decimal.NewFromInt(10), Direction: Debit},
				{Amount: decimal.NewFromInt(9), Direction: Credit},
			},
			want: false,
		},
		{
			name:    "empty transaction is rejected",
			entries: nil,
			want:    false,
		},
		{
			name: "unknown direction is rejected",
			entries: []Entry{
				{Amount: decimal.NewFromInt(1), Direction: Debit},
				{Amount: decimal.NewFromInt(1), Direction: "credit"},
			},
			want: false,
		},
		{
			name: "zero amount is rejected",
			entries: []Entry{
				{Amount: decimal.Zero, Direction: Debit},
				{Amount: decimal.Zero, Direction: Credit},
			},
			want: false,
		},
		{
			name: "mixed currencies are rejected",
			entries: []Entry{
				{Amount: decimal.NewFromInt(1), Direction: Debit, Currency: "USD"},
				{Amount: decimal.NewFromInt(1), Direction: Credit, Currency: "EUR"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := range tt.entries {
				tt.entries[i].AccountID = uuid.New()
				if tt.entries[i].Currency == "" {
					tt.entries[i].Currency = "USD"
				}
			}
			tx := Transaction{Entries: tt.entries}
			if got := tx.IsBalanced(); got != tt.want {
				t.Fatalf("IsBalanced() = %v, want %v", got, tt.want)
			}
		})
	}
}
