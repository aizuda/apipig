package initialize

import (
	aiModel "apipig/app/ai/model"
	reviewModel "apipig/app/apps/code-review/model"
	sysModel "apipig/app/sys/model"
	"log"

	"gorm.io/gorm"
)

func autoMigrate(db *gorm.DB) error {
	log.Println("Initiating migration...")
	if err := db.AutoMigrate(
		&aiModel.Provider{},
		&aiModel.Channel{},
		&aiModel.ChannelAccount{},
		&aiModel.AccessToken{},
		&aiModel.AccessTokenTag{},
		&aiModel.AccessTokenTagRelation{},
		&aiModel.Proxy{},
		&aiModel.CallLog{},
		&reviewModel.Project{},
		&reviewModel.Task{},
		&sysModel.Resource{},
		&sysModel.ResourceApi{},
		&sysModel.Role{},
		&sysModel.RoleResource{},
		&sysModel.User{},
		&sysModel.UserRole{},
		&sysModel.UserSession{},
	); err != nil {
		return err
	}
	log.Println("Migration Completed...")
	return nil
}
