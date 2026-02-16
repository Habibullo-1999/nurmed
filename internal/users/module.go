package users

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/fx"

	intauth "nurmed/internal/auth"
	"nurmed/internal/interfaces"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger   logger.ILogger
	UserRepo interfaces.UserRepo
	AuthSvs  intauth.Service
}

type service struct {
	logger   logger.ILogger
	userRepo interfaces.UserRepo
	authSvs  intauth.Service
}

type Service interface {
	GetUsers(ctx context.Context, request structs.UserFilter) ([]structs.UserResponse, error)
	CreateUser(ctx context.Context, request structs.CreateUserRequest) (structs.UserResponse, error)
}

var (
	ErrInvalidUserPayload = errors.New("invalid user payload")
	ErrConflict           = errors.New("conflict")
)

func New(p Params) Service {
	return &service{
		logger:   p.Logger,
		userRepo: p.UserRepo,
		authSvs:  p.AuthSvs,
	}
}

func (s *service) GetUsers(ctx context.Context, request structs.UserFilter) ([]structs.UserResponse, error) {

	var usersResponse []structs.UserResponse
	request.Validate()

	users, err := s.userRepo.GetUsers(ctx, request)
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		usersResponse = append(usersResponse, structs.UserResponse{
			ID:          user.ID,
			CompanyID:   user.CompanyID,
			UserName:    user.UserName,
			Phone:       user.Phone,
			Email:       user.Email,
			Status:      user.Status,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			LastLoginAt: user.LastLoginAt,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		})
	}

	return usersResponse, nil
}

func (s *service) CreateUser(ctx context.Context, request structs.CreateUserRequest) (structs.UserResponse, error) {
	request.UserName = strings.TrimSpace(request.UserName)
	request.Phone = strings.TrimSpace(request.Phone)
	request.Email = strings.TrimSpace(request.Email)
	request.FirstName = strings.TrimSpace(request.FirstName)
	request.LastName = strings.TrimSpace(request.LastName)
	request.Status = strings.ToLower(strings.TrimSpace(request.Status))
	request.Role.RoleCode = strings.TrimSpace(request.Role.RoleCode)
	request.Role.ScopeType = strings.ToLower(strings.TrimSpace(request.Role.ScopeType))

	if request.UserName == "" || request.Password == "" || request.FirstName == "" || request.Role.RoleCode == "" {
		return structs.UserResponse{}, ErrInvalidUserPayload
	}
	if request.Status == "" {
		request.Status = structs.UserStatusActive
	}
	if !isValidStatus(request.Status) {
		return structs.UserResponse{}, ErrInvalidUserPayload
	}

	if request.Role.ScopeType == "" {
		if request.IsSuperAdmin {
			request.Role.ScopeType = "global"
		} else {
			request.Role.ScopeType = "company"
		}
	}
	if request.Role.ScopeType == "company" && request.Role.ScopeID == nil {
		if request.CompanyID == 0 {
			return structs.UserResponse{}, ErrInvalidUserPayload
		}
		scopeID := request.CompanyID
		request.Role.ScopeID = &scopeID
	}
	if request.Role.ScopeType == "global" {
		request.Role.ScopeID = nil
	}
	if !isValidScopeType(request.Role.ScopeType) {
		return structs.UserResponse{}, ErrInvalidUserPayload
	}
	if request.Role.ScopeType != "global" && request.Role.ScopeID == nil {
		return structs.UserResponse{}, ErrInvalidUserPayload
	}

	passwordHash, err := s.authSvs.HashPassword(request.Password)
	if err != nil {
		return structs.UserResponse{}, err
	}

	userEntity := structs.User{
		CompanyID:    request.CompanyID,
		UserName:     request.UserName,
		Phone:        request.Phone,
		Email:        request.Email,
		PasswordHash: passwordHash,
		FirstName:    request.FirstName,
		LastName:     request.LastName,
		Status:       request.Status,
		IsSuperAdmin: request.IsSuperAdmin,
	}

	createdUser, err := s.userRepo.CreateUserWithRoleScope(ctx, userEntity, request.Role)
	if err != nil {
		return structs.UserResponse{}, err
	}

	return structs.UserResponse{
		ID:          createdUser.ID,
		CompanyID:   createdUser.CompanyID,
		UserName:    createdUser.UserName,
		Phone:       createdUser.Phone,
		Email:       createdUser.Email,
		FirstName:   createdUser.FirstName,
		LastName:    createdUser.LastName,
		Status:      createdUser.Status,
		LastLoginAt: createdUser.LastLoginAt,
		CreatedAt:   createdUser.CreatedAt,
		UpdatedAt:   createdUser.UpdatedAt,
	}, nil
}

func isValidStatus(status string) bool {
	switch status {
	case "active", "blocked", "invited", "deleted":
		return true
	default:
		return false
	}
}

func isValidScopeType(scopeType string) bool {
	switch scopeType {
	case "global", "company", "branch", "warehouse":
		return true
	default:
		return false
	}
}
