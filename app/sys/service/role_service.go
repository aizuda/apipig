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
	"apipig/toolkit/snowflake"
	"errors"

	"gorm.io/gorm"
)

type RoleService struct {
}

func (s *RoleService) getDB() *gorm.DB {
	return db.GetDB(model.Role{})
}

func (s *RoleService) ResourceSet(params *sysReq.RoleResourceSetParams) (bool, error) {
	create := params.ID == 0

	// 新增角色判断是否存在
	if create {
		if s.ExistByUsername(params.Name) {
			return false, errors.New("角色名称已存在")
		}
	} else {
		if params.ID == 0 {
			return false, errors.New("角色ID不能为空")
		}
	}

	role := model.Role{
		Name:   params.Name,
		Alias:  params.Alias,
		Remark: params.Remark,
		Status: api.NormalDisable(params.Status),
		Sort:   params.Sort,
	}

	err := db.Transaction(func(tx *gorm.DB) error {

		if create {
			// 创建角色基本信息
			role.MODEL = db.NewModel(params.Ctx)
		} else {
			// 设置角色ID
			role.ID = params.ID
			// 删除历史关联
			tx.Delete(model.RoleResource{}, "role_id=?", role.ID)
		}
		if len(params.ResourceIds) > 0 {
			for _, resourceId := range params.ResourceIds {
				if err := tx.Create(&model.RoleResource{
					ID:         db.GetId(),
					RoleId:     role.ID,
					ResourceId: resourceId,
				}).Error; err != nil {
					return err
				}
			}
		}
		if create {
			// 保存角色
			if err := tx.Create(&role).Error; err != nil {
				return err
			}
			return nil
		}

		// 根据ID更新角色
		role.ID = params.ID
		return tx.Updates(role).Error
	})

	return err == nil, err
}

func (s *RoleService) ExistByUsername(name string) bool {
	return db.Exist(model.Role{}, func(t *gorm.DB) *gorm.DB {
		return t.Where("name=?", name)
	})
}

func (s *RoleService) Delete(idsReq *request.IdsReq) (bool, error) {
	return db.DeleteByIds(model.Role{}, idsReq.Ids)
}

func (s *RoleService) Update(m *model.Role) (bool, error) {
	if m.Status != 0 {
		m.Status = api.NormalDisable(m.Status)
	}
	return db.Update(m)
}

func (s *RoleService) GetRoleRespById(id snowflake.ID) (roleResp sysResp.RoleResp, err error) {
	err = db.GetById(&roleResp.Role, id)
	if err == nil {
		global.DB.Raw("SELECT resource_id FROM ap_role_resource WHERE role_id=?",
			id).Find(&roleResp.ResourceIds)
	}
	return
}

func (s *RoleService) List(params *sysReq.RoleParams) (arr []model.Role, err error) {
	t := s.getDB().Where("status=?", 1)
	if len(params.Name) > 0 {
		t.Where("name like ?", "%"+params.Name+"%")
	}
	err = t.Find(&arr).Error
	return
}

func (s *RoleService) Page(params *sysReq.RolePageParams) (response.PageResult, error) {
	t := s.getDB()
	if len(params.Name) > 0 {
		t.Where("name like ?", "%"+params.Name+"%")
	}
	if len(params.Alias) > 0 {
		t.Where("alias like ?", "%"+params.Alias+"%")
	}
	if params.Status > 0 {
		t.Where("status = ?", params.Status)
	}
	t.Order("sort DESC")
	var arr []model.Role
	return db.Page(t, params.PageInfo, arr)
}
