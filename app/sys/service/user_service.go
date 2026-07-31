package service

import (
	"apipig/app/sys/model"
	sysReq "apipig/app/sys/model/request"
	sysResp "apipig/app/sys/model/response"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/middleware"
	"apipig/toolkit"
	"apipig/toolkit/snowflake"
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserService struct {
}

func (s *UserService) getDB() *gorm.DB {
	return db.GetDB(model.User{})
}

func (s *UserService) AssignSet(params *sysReq.UserAssignSetParams) (bool, error) {
	create := params.ID == 0

	// 新增用户判断是否存在
	if create {
		if len(params.Password) == 0 {
			return false, errors.New("用户密码不能为空")
		}
		if s.ExistByUsername(params.Username) {
			return false, errors.New("用户名已存在")
		}
	} else {
		if params.ID == 0 {
			return false, errors.New("用户ID不能为空")
		}
	}

	user := model.User{
		Username: params.Username,
		RealName: params.RealName,
		NickName: params.NickName,
		Avatar:   params.Avatar,
		Sex:      params.Sex,
		Phone:    params.Phone,
		Email:    params.Email,
		JobNum:   params.JobNum,
	}

	err := db.Transaction(func(tx *gorm.DB) error {

		if create {
			// 创建用户基本信息
			user.MODEL = db.NewModel(params.Ctx)
			// 设置登录密码
			user.Password = toolkit.GetPassword(params.Password, params.Username)
		} else {
			// 设置用户ID
			user.ID = params.ID
			// 删除历史关联
			tx.Delete(model.UserRole{}, "user_id=?", user.ID)
		}
		if len(params.RoleIds) > 0 {
			for _, roleId := range params.RoleIds {
				if err := tx.Create(&model.UserRole{
					ID:     db.GetId(),
					UserId: user.ID,
					RoleId: roleId,
				}).Error; err != nil {
					return err
				}
			}
		}
		if create {
			// 保存用户
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
			return nil
		}

		// 根据ID更新用户
		user.ID = params.ID
		return tx.Updates(user).Error
	})

	return err == nil, err
}

// CreateSession 创建用户登录会话信息
func (s *UserService) CreateSession(c *fiber.Ctx, userId snowflake.ID, username, sid string) (bool, error) {
	nowTime := toolkit.GetNowUnixMilli()
	return db.Create(&model.UserSession{
		ID:        db.GetId(),
		CreatedAt: nowTime,
		UserId:    userId,
		Username:  username,
		Sid:       sid,
		Browser:   c.Get("User-Agent"),
		Ip:        c.IP(),
		St:        nowTime,
	})
}

func (s *UserService) ExistByUsername(username string) bool {
	return db.Exist(model.User{}, func(t *gorm.DB) *gorm.DB {
		return t.Where("username=?", username)
	})
}

func (s *UserService) Delete(idsReq *request.IdsReq) (bool, error) {
	return db.DeleteByIds(model.User{}, idsReq.Ids)
}

func (s *UserService) DeleteExpiredSession() error {
	ts := global.CONFIG.JWT.GetExpiresTime()
	return global.DB.Where("st < ?", ts).Delete(&model.UserSession{}).Error
}

func (s *UserService) Update(m *model.User) (bool, error) {
	if m.Status != 0 {
		m.Status = api.NormalDisable(m.Status)
	}
	return db.Update(m)
}

// UpdateSession 更新用户登录会话时间
func (s *UserService) UpdateSession(userId snowflake.ID, sid, ip string) error {
	t := global.DB.Model(&model.UserSession{})
	if global.CONFIG.JWT.IgnoreIp {
		t.Where("user_id=? AND sid=?", userId, sid)
	} else {
		t.Where("user_id=? AND sid=? AND ip=?", userId, sid, ip)
	}
	err := t.Updates(&model.UserSession{St: toolkit.GetNowUnixMilli()}).Error
	if err != nil {
		global.LOG.Error("illegal login", zap.String("userId", userId.String()),
			zap.String("sid", sid), zap.String("ip", ip))
	}
	return err
}

func (s *UserService) ResetPassword(params *sysReq.ResetPasswordParams) (bool, error) {
	var userList []model.User
	err := db.GetByIds(&userList, params.Ids)
	if err != nil {
		return false, err
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, user := range userList {
			_, err := db.Update(model.User{
				MODEL: api.MODEL{
					ID: user.ID,
				},
				Password: toolkit.GetPassword(params.Password, user.Username),
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err == nil, err
}

func (s *UserService) AssignRoles(params *sysReq.AssignRolesParams) (bool, error) {
	var userRoles []model.UserRole
	for _, userId := range params.UserIds {
		for _, roleId := range params.RoleIds {
			userRoles = append(userRoles, model.UserRole{
				ID:     db.GetId(),
				UserId: userId,
				RoleId: roleId,
			})
		}
	}
	return db.Create(userRoles)
}

func (s *UserService) GetById(id snowflake.ID) (m model.User, err error) {
	err = db.GetById(&m, id)
	return
}

func (s *UserService) GetUserRespById(id snowflake.ID) (userResp sysResp.UserResp, err error) {
	err = db.GetById(&userResp.User, id)
	if err == nil {
		global.DB.Raw("SELECT role_id FROM ap_user_role WHERE user_id=?",
			id).Find(&userResp.RoleIds)
		global.DB.Raw("SELECT department_id FROM ap_user_department WHERE user_id=?",
			id).Find(&userResp.DepartmentIds)
	}
	return
}

func (s *UserService) GetUserInfo(c *fiber.Ctx) (user model.User, err error) {
	tc := middleware.GetTokenClaims(c)
	if tc != nil {
		err = db.GetById(&user, tc.ID)
		if err == nil {
			// 不显示密码
			user.Password = ""
		}
	}
	return
}

func (s *UserService) GetByUsername(username string) (user model.User, err error) {
	err = s.getDB().Where("username=?", username).First(&user).Error
	return
}

func (s *UserService) List(params *sysReq.UserParams) (userList []model.User, err error) {
	err = s.getDB().Where("status=?", 1).Find(&userList).Error
	return
}

func (s *UserService) Page(params *sysReq.UserPageParams) (response.PageResult, error) {
	t := s.getDB()
	if len(params.Username) > 0 {
		t.Where("username like ?", "%"+params.Username+"%")
	}
	if len(params.RealName) > 0 {
		t.Where("real_name like ?", "%"+params.RealName+"%")
	}
	if len(params.NickName) > 0 {
		t.Where("nick_name like ?", "%"+params.NickName+"%")
	}
	if params.Sex > 0 {
		t.Where("sex = ?", params.Sex)
	}
	if len(params.Phone) > 0 {
		t.Where("phone like ?", "%"+params.Phone+"%")
	}
	if len(params.Email) > 0 {
		t.Where("email like ?", "%"+params.Email+"%")
	}
	if params.Status > 0 {
		t.Where("status = ?", params.Status)
	}
	if len(params.JobNum) > 0 {
		t.Where("job_num like ?", "%"+params.JobNum+"%")
	}
	if params.DepartmentId > 0 {
		t.Where(`id IN (WITH RECURSIVE r AS (SELECT id FROM ap_department WHERE id=?
        UNION ALL SELECT c.id FROM ap_department c JOIN r ON c.pid=r.id AND c.deleted_at=0)
        SELECT DISTINCT d.user_id FROM ap_user_department d JOIN r ON d.department_id=r.id)
        `, params.DepartmentId)
	}
	var arr []model.User
	return db.Page(t, params.PageInfo, arr)
}
