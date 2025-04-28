package interfaces

import (
	"context"

	"github.com/erinchen11/hr-system/internal/models"
)

type AuthService interface {
	Authenticate(ctx context.Context, email, password string) (*models.Account, error)
}
