package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

const (
	ProjectStatusEnabled  uint = 1
	ProjectStatusDisabled uint = 2
)

// Project 定义一个接收 Git WebHook 并执行 AI 代码评审的仓库配置。
type Project struct {
	api.MODEL
	Name               string       `gorm:"size:100;not null;index" json:"name"`
	Provider           string       `gorm:"size:20;not null;default:github" json:"provider"`
	RepositoryURL      string       `gorm:"size:1000;not null" json:"repositoryUrl"`
	DefaultBranch      string       `gorm:"size:200;not null;default:main" json:"defaultBranch"`
	WebhookKey         string       `gorm:"size:80;not null;uniqueIndex" json:"webhookKey"`
	WebhookSecret      string       `gorm:"size:1000;not null" json:"-"`
	RepositoryToken    string       `gorm:"type:text" json:"-"`
	AccessTokenID      snowflake.ID `gorm:"type:bigint;not null;index" json:"accessTokenId" swaggertype:"string"`
	Model              string       `gorm:"size:200;not null" json:"model"`
	ReviewPrompt       string       `gorm:"type:text" json:"reviewPrompt"`
	BranchPattern      string       `gorm:"size:500" json:"branchPattern"`
	IgnorePatterns     string       `gorm:"type:text" json:"ignorePatterns"`
	PushEnabled        bool         `gorm:"not null;default:true" json:"pushEnabled"`
	PullRequestEnabled bool         `gorm:"not null;default:true" json:"pullRequestEnabled"`
	Status             uint         `gorm:"type:smallint;not null;default:1;index" json:"status"`
	Remark             string       `gorm:"size:500" json:"remark"`
}

func (Project) TableName() string { return "ap_review_project" }
