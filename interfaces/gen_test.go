package interfaces

import (
	"log"
	"testing"
)

func TestGen(t *testing.T) {
	// Example usage
	interfaceSrc := `
package example

import (
	"context"
)

type RBAC interface {
	GetAccountsForUser(ctx context.Context, userID string) ([]*AccountUserRole, error)

	NewAccountUserRole(ctx context.Context, accountID string, roleID string, userID string) (*AccountUserRole, error)
	DeleteAccountUserRole(ctx context.Context, accountID string, roleID string, userID string) error
}
`

	err := GenerateHTTPHandlers(interfaceSrc, "handlers", "./generated")
	if err != nil {
		log.Fatalf("Error generating handlers: %v", err)
	}
}
