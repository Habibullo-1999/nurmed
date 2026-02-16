package interfaces

import (
	"context"

	"nurmed/internal/structs"
)

type UserRepo interface {
	GetUsers(ctx context.Context, request structs.UserFilter) ([]structs.User, error)
	CreateUserWithRoleScope(ctx context.Context, user structs.User, role structs.UserRoleAssignment) (structs.User, error)
}
