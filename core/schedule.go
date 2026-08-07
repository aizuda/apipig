package core

import (
	sys "apipig/app/sys/service"
	"apipig/global"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func RunJobs() {
	job := cron.New()

	// 定时删除在线会话过期用户，每5分钟检查一次
	_, err := job.AddFunc(global.CONFIG.System.SessionCron, func() {
		err := sys.SysService.UserService.DeleteExpiredSession()
		if err != nil {
			global.LOG.Error("delete user session failed", zap.Any("err", err))
		}
	})
	if err != nil {
		panic(err)
	}

	// 启动定时任务
	job.Start()
}
