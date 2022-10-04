package middle

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestAuthSignature_Valid(t *testing.T) {
	id := uuid.New().String()
	key := uuid.New().String()
	authSignature := &AuthSignature{
		ID:  id,
		Key: key,
	}
	authSignature.computeSignature(5*time.Minute, "12345678", "12345678")
	println(authSignature.Signature)
	time.Sleep(1 * time.Second)
	authSignature2 := &AuthSignature{
		ID:  id,
		Key: key,
	}
	authSignature2.computeSignature(5*time.Minute, "12345678", "12345678")
	println(authSignature2.Signature)
}
