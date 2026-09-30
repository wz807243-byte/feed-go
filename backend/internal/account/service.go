package account

import (
	"context"
	"feedsystem_video_go/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

type AccountService struct {
	accountRepository *AccountRepository
}

func NewAccountService(accountRepository *AccountRepository) *AccountService {
	return &AccountService{accountRepository: accountRepository}

}
func (as *AccountService) CreateAccount(ctx context.Context, account *Account) error {
	passwordHars, err := bcrypt.GenerateFromPassword([]byte(account.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	account.Password = string(passwordHars)
	if err := as.accountRepository.CreateAccount(ctx, account); err != nil {
		return err
	}
	return nil
}
func (as *AccountService) FindByUsername(ctx context.Context, username string) (*Account, error) {
	account, err := as.accountRepository.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return account, nil
}
func (as *AccountService) FindByID(ctx context.Context, id uint) (*Account, error) {
	account, err := as.accountRepository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return account, nil
}
func (as *AccountService) Login(ctx context.Context, username string, password string) (string, string, error) {
	account, err := as.FindByUsername(ctx, username)
	if err != nil {
		return "", "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password)); err != nil {
		return "", "", err
	}
	accessToken, err := auth.GenerateToken(account.ID, account.Username)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := auth.GenerateRefreshToken(account.ID)
	if err != nil {
		return "", "", err
	}
	if err := as.accountRepository.Login(ctx, account.ID, accessToken, refreshToken); err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}
func (as *AccountService) Logout(ctx context.Context, accountID uint) error {
	account, err := as.FindByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.Token == "" {
		return err
	}
	return as.accountRepository.Logout(ctx, account.ID)
}
