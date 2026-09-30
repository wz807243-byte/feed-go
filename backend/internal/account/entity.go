package account

type Account struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"unique" json:"username"`
	Password     string `json:"-"`
	Token        string `json:"-"`
	RefreshToken string `json:"-"`
}

type CreateAccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type FindByIDRequest struct {
	ID uint `json:"id"`
}

type FindByUsernameRequest struct {
	Username string `json:"username"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    uint   `json:"account_id"`
	Username     string `json:"username"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
