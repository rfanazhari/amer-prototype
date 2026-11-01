package accessrepository

import "time"

// AccountModel maps administrative_accounts independently of domain and API types.
type AccountModel struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	Name         string `gorm:"type:text;not null"`
	Email        string `gorm:"type:text;not null"`
	PasswordHash string `gorm:"column:password_hash;type:text;not null"`
	Role         string `gorm:"type:text;not null"`
	Status       string `gorm:"type:text;not null"`
	CreatedBy    *string
	CreatedAt    time.Time `gorm:"not null;autoCreateTime:false"`
	UpdatedAt    time.Time `gorm:"not null;autoUpdateTime:false"`
}

// TableName fixes the administrative account table used by GORM.
func (AccountModel) TableName() string { return "administrative_accounts" }

// SessionModel maps administrative_sessions independently of domain and API types.
type SessionModel struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	AccountID string    `gorm:"column:account_id;type:uuid;not null"`
	TokenHash string    `gorm:"column:token_hash;type:text;not null"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime:false"`
	ExpiresAt time.Time `gorm:"not null"`
	RevokedAt *time.Time
}

// TableName fixes the administrative session table used by GORM.
func (SessionModel) TableName() string { return "administrative_sessions" }
