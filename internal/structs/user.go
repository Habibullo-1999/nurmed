package structs

import "time"

type User struct {
	ID               int64
	CompanyID        int64
	UserName         string
	Phone            string
	Email            string
	PasswordHash     string
	FirstName        string
	LastName         string
	Status           string
	IsSuperAdmin     bool
	FailedLoginCount int
	LockedUntil      *time.Time
	LastLoginAt      time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UserRequest struct {
	ID           int64  `json:"id"`
	CompanyID    int64  `json:"companyID"`
	UserName     string `json:"userName"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName,omitempty"`
	Status       string `json:"status"`
	IsSuperAdmin bool   `json:"isSuperAdmin"`
}

type UserRoleAssignment struct {
	RoleCode  string `json:"roleCode" binding:"required"`
	ScopeType string `json:"scopeType"`
	ScopeID   *int64 `json:"scopeId,omitempty"`
	OwnOnly   bool   `json:"ownOnly"`
}

type CreateUserRequest struct {
	CompanyID    int64              `json:"companyId"`
	UserName     string             `json:"userName" binding:"required"`
	Phone        string             `json:"phone"`
	Email        string             `json:"email"`
	Password     string             `json:"password" binding:"required"`
	FirstName    string             `json:"firstName" binding:"required"`
	LastName     string             `json:"lastName"`
	Status       string             `json:"status"`
	IsSuperAdmin bool               `json:"isSuperAdmin"`
	Role         UserRoleAssignment `json:"role" binding:"required"`
}

type UserResponse struct {
	ID          int64     `json:"id"`
	CompanyID   int64     `json:"companyID"`
	UserName    string    `json:"userName"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	FirstName   string    `json:"firstName"`
	LastName    string    `json:"lastName"`
	Status      string    `json:"status"`
	LastLoginAt time.Time `json:"lastLoginAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type UserFilter struct {
	ID           int64  `form:"id" json:"id,omitempty"`
	CompanyID    int64  `form:"company_id" json:"companyID,omitempty"`
	UserName     string `form:"username" json:"userName,omitempty"`
	Status       string `form:"status" json:"status,omitempty"`
	Phone        string `form:"phone" json:"phone,omitempty"`
	Email        string `form:"email" json:"email,omitempty"`
	FirstName    string `form:"first_name" json:"firstName,omitempty"`
	LastName     string `form:"last_name" json:"lastName,omitempty"`
	IsSuperAdmin bool   `form:"is_super_admin" json:"isSuperAdmin,omitempty"`
	Pagination
}
