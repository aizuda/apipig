package service

import (
	"errors"
	"fmt"

	"apipig/app/ai/model"

	"gorm.io/gorm"
)

// MigrateLegacyAccessTokenData 在启动时迁移历史明文 Token，并回填整数金额账本。
func MigrateLegacyAccessTokenData(database *gorm.DB) error {
	if database == nil {
		return errors.New("迁移 API Token 时数据库不能为空")
	}
	var tokens []model.AccessToken
	if err := database.Unscoped().Select("id", "token", "quota_amount", "used_amount", "quota_micro_usd", "used_micro_usd").
		Find(&tokens).Error; err != nil {
		return err
	}
	return database.Transaction(func(transaction *gorm.DB) error {
		for _, token := range tokens {
			updates := make(map[string]any, 3)
			if token.Token != "" && !isHashedGatewayToken(token.Token) {
				updates["token"] = hashGatewayToken(token.Token)
			}
			if token.QuotaMicroUSD == 0 && token.QuotaAmount > 0 {
				updates["quota_micro_usd"] = usdToMicroUSD(token.QuotaAmount)
			}
			if token.UsedMicroUSD == 0 && token.UsedAmount > 0 {
				updates["used_micro_usd"] = usdToMicroUSD(token.UsedAmount)
			}
			if len(updates) == 0 {
				continue
			}
			if err := transaction.Unscoped().Model(&model.AccessToken{}).Where("id = ?", token.ID).
				Updates(updates).Error; err != nil {
				return fmt.Errorf("迁移 API Token %s 失败: %w", token.ID.String(), err)
			}
		}
		return nil
	})
}
