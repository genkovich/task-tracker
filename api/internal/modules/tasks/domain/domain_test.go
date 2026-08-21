package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

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
			require.ErrorIs(t, tc.card.Validate(), domain.ErrNameRequired)
		})
	}
}

func TestCard_Validate_NameTooLong(t *testing.T) {
	card := domain.Card{Name: strings.Repeat("a", 201)}
	require.ErrorIs(t, card.Validate(), domain.ErrCardFieldTooLong)
}

func TestCard_Validate_AssigneeTooLong(t *testing.T) {
	assignee := strings.Repeat("a", 101)
	card := domain.Card{Name: "ok", Assignee: &assignee}
	require.ErrorIs(t, card.Validate(), domain.ErrCardFieldTooLong)
}

func TestCard_Validate_OK(t *testing.T) {
	assignee := "Test User"
	card := domain.Card{Name: "Write the deck", Assignee: &assignee, ColumnStatus: domain.ColumnTodo}
	require.NoError(t, card.Validate())
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
		t.Run(tc.status, func(t *testing.T) {
			require.Equal(t, tc.valid, domain.IsValidColumnStatus(tc.status))
		})
	}
}
