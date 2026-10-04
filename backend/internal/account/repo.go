package account

import (
	"context"

	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (ar *AccountRepository) CreateAccount(ctx context.Context, account *Account) error {
	return ar.db.WithContext(ctx).Create(account).Error
}

func (ar *AccountRepository) FindByID(ctx context.Context, id uint) (*Account, error) {
	var account Account
	if err := ar.db.WithContext(ctx).First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (ar *AccountRepository) FindByUsername(ctx context.Context, username string) (*Account, error) {
	var account Account
	if err := ar.db.WithContext(ctx).Where("username = ?", username).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (ar *AccountRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*Account, error) {
	var account Account
	if err := ar.db.WithContext(ctx).Where("refresh_token = ?", refreshToken).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (ar *AccountRepository) Login(ctx context.Context, id uint, token, refreshToken string) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{"token": token, "refresh_token": refreshToken}).Error
}

func (ar *AccountRepository) Logout(ctx context.Context, id uint) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{"token": "", "refresh_token": ""}).Error
}

func (ar *AccountRepository) UpdateToken(ctx context.Context, id uint, token string) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("token", token).Error
}
func (ar *AccountRepository) UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id=?", id).Updates(updates).Error
}

func (ar *AccountRepository) UpdateAvatar(ctx context.Context, accountID uint, avatarURL string) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", accountID).Update("avatar_url", avatarURL).Error
}
func (ar *AccountRepository) Rename(ctx context.Context, accountID uint, newUsername string) {

}
func (ar *AccountRepository) RenameWithToken(ctx context.Context, accountID uint, newUsername string, token string) error {
	return ar.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&Account{}).Where("id=?", accountID).Update("username", newUsername)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound

		}
		if err := tx.Model(&Account{}).Where("id=?", accountID).Update("username", newUsername).Error; err != nil {
			return err
		}
		return nil
	})
}
func (ar *AccountRepository) ChangePassword(ctx context.Context, id uint, newPassword string) error {
	err := ar.db.WithContext(ctx).Model(&Account{}).Where("id=?", id).Update("password", newPassword).Error
	if err != nil {
		return err
	}
	return err
}
