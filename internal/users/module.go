package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/fx"

	intauth "nurmed/internal/auth"
	"nurmed/internal/interfaces"
	"nurmed/internal/structs"
	"nurmed/pkg/ctxutil"
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

var ErrNotFound = errors.New("not found")

type Service interface {
	GetUsers(ctx context.Context, request structs.UserFilter) ([]structs.UserResponse, error)
	CreateUser(ctx context.Context, request structs.CreateUserRequest) (structs.UserResponse, error)
	UpdateUser(ctx context.Context, id int64, req structs.UpdateUserRequest) (structs.UserResponse, error)
	DeleteUser(ctx context.Context, id int64) error
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

	resourceID := fmt.Sprintf("%d", createdUser.ID)
	s.authSvs.Audit(ctx, structs.AuditLog{
		UserID:     ctxutil.ActorID(ctx),
		Action:     "user.create",
		Module:     "admin",
		Resource:   "user",
		ResourceID: &resourceID,
		Meta: map[string]interface{}{
			"userName":  createdUser.UserName,
			"companyId": createdUser.CompanyID,
			"role":      request.Role.RoleCode,
		},
	})

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

func (s *service) UpdateUser(ctx context.Context, id int64, req structs.UpdateUserRequest) (structs.UserResponse, error) {
	if req.Status != "" && !isValidStatus(req.Status) {
		return structs.UserResponse{}, ErrInvalidUserPayload
	}

	updated, err := s.userRepo.UpdateUser(ctx, id, req)
	if err != nil {
		return structs.UserResponse{}, err
	}

	resourceID := fmt.Sprintf("%d", id)
	meta := map[string]interface{}{}
	if req.FirstName != "" {
		meta["firstName"] = req.FirstName
	}
	if req.LastName != "" {
		meta["lastName"] = req.LastName
	}
	if req.Status != "" {
		meta["status"] = req.Status
	}
	s.authSvs.Audit(ctx, structs.AuditLog{
		UserID:     ctxutil.ActorID(ctx),
		Action:     "user.update",
		Module:     "admin",
		Resource:   "user",
		ResourceID: &resourceID,
		Meta:       meta,
	})

	return structs.UserResponse{
		ID:          updated.ID,
		CompanyID:   updated.CompanyID,
		UserName:    updated.UserName,
		Phone:       updated.Phone,
		Email:       updated.Email,
		FirstName:   updated.FirstName,
		LastName:    updated.LastName,
		Status:      updated.Status,
		LastLoginAt: updated.LastLoginAt,
		CreatedAt:   updated.CreatedAt,
		UpdatedAt:   updated.UpdatedAt,
	}, nil
}

func (s *service) DeleteUser(ctx context.Context, id int64) error {
	if err := s.userRepo.DeleteUser(ctx, id); err != nil {
		return err
	}

	resourceID := fmt.Sprintf("%d", id)
	s.authSvs.Audit(ctx, structs.AuditLog{
		UserID:     ctxutil.ActorID(ctx),
		Action:     "user.delete",
		Module:     "admin",
		Resource:   "user",
		ResourceID: &resourceID,
		Meta:       map[string]interface{}{},
	})

	return nil
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
