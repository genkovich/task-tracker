package domain_test

import (
	"strings"
	"testing"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/domain"
)

func TestCard_Validate_NameRequired(t *testing.T) {
	cases := []struct {
		name string
		card domain.Card
	}{
		{"empty name", domain.Card{Name: ""}},
		{"whitespace-only name", domain.Card{Name: "   "}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.card.Validate(); err != domain.ErrNameRequired {
				t.Fatalf("Validate() = %v, want %v", err, domain.ErrNameRequired)
			}
		})
	}
}

func TestCard_Validate_NameTooLong(t *testing.T) {
	card := domain.Card{Name: strings.Repeat("a", 201)}
	if err := card.Validate(); err != domain.ErrCardFieldTooLong {
		t.Fatalf("Validate() = %v, want %v", err, domain.ErrCardFieldTooLong)
	}
}

func TestCard_Validate_AssigneeTooLong(t *testing.T) {
	assignee := strings.Repeat("a", 101)
	card := domain.Card{Name: "ok", Assignee: &assignee}
	if err := card.Validate(); err != domain.ErrCardFieldTooLong {
		t.Fatalf("Validate() = %v, want %v", err, domain.ErrCardFieldTooLong)
	}
}

func TestCard_Validate_OK(t *testing.T) {
	assignee := "Test User"
	card := domain.Card{Name: "Write the deck", Assignee: &assignee, ColumnStatus: domain.ColumnTodo}
	if err := card.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestIsValidColumnStatus(t *testing.T) {
	cases := []struct {
		status string
		valid  bool
	}{
		{domain.ColumnTodo, true},
		{domain.ColumnInProgress, true},
		{domain.ColumnDone, true},
		{"backlog", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := domain.IsValidColumnStatus(tc.status); got != tc.valid {
			t.Errorf("IsValidColumnStatus(%q) = %v, want %v", tc.status, got, tc.valid)
		}
	}
}
