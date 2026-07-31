package service

import (
	"apipig/app/sys/model"
	sysReq "apipig/app/sys/model/request"
	sysResp "apipig/app/sys/model/response"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/db"
	"apipig/middleware"
	"apipig/toolkit/snowflake"
	"errors"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ResourceService struct {
}

func (s *ResourceService) getDB() *gorm.DB {
	return db.GetDB(model.Resource{})
}

func (s *ResourceService) Create(params *sysReq.ResourceCreateParams) (bool, error) {
	return db.Create(&model.Resource{
		MODEL:       db.NewModel(params.Ctx),
		Pid:         params.Pid,
		Title:       params.Title,
		Alias:       params.Alias,
		Type:        params.Type,
		Code:        params.Code,
		Redirect:    params.Redirect,
		Path:        params.Path,
		Icon:        params.Icon,
		Status:      api.NormalDisable(params.Status),
		Sort:        params.Sort,
		Component:   params.Component,
		Color:       params.Color,
		Hidden:      params.Hidden,
		ParentRoute: params.ParentRoute,
		KeepAlive:   params.KeepAlive,
		Query:       params.Query,
	})
}

func (s *ResourceService) Delete(idsReq *request.IdsReq) (bool, error) {
	if db.ExistInPid(model.Resource{}, idsReq.Ids) {
		return false, errors.New("存在子菜单不允许删除")
	}
	if db.Exist(model.RoleResource{}, func(t *gorm.DB) *gorm.DB {
		return t.Where("resource_id IN (?)", idsReq.Ids)
	}) {
		return false, errors.New("存在角色关联菜单不允许删除")
	}
	if db.Exist(model.ResourceApi{}, func(t *gorm.DB) *gorm.DB {
		return t.Where("resource_id IN (?)", idsReq.Ids)
	}) {
		return false, errors.New("存在关联资源权限API不允许删除")
	}
	return db.DeleteByIds(model.Resource{}, idsReq.Ids)
}

func (s *ResourceService) Update(m *model.Resource) (bool, error) {
	if m.Status != 0 {
		m.Status = api.NormalDisable(m.Status)
	}
	return db.Update(m)
}

func (s *ResourceService) GetById(id snowflake.ID) (m model.Resource, err error) {
	err = db.GetById(&m, id)
	return
}

func (s *ResourceService) ListMenu(c *fiber.Ctx) (resp sysResp.ResourceMenuResp, err error) {
	var resources []model.Resource
	tc := middleware.GetTokenClaims(c)
	err = s.getDB().Where("status=1 AND type IN (1,2,3) AND id IN (SELECT c.resource_id FROM ap_role_resource c JOIN ap_role r ON c.role_id=r.id JOIN ap_user_role u ON r.id=u.role_id WHERE u.user_id=?)",
		tc.ID).Order("sort DESC").Find(&resources).Error
	if err == nil && len(resources) > 0 {
		resp.Menus = getMenus(snowflake.ParseInt64(1), resources)
		resp.Permissions = []string{}
	}
	return
}

// 递归子节点
func getMenus(id snowflake.ID, resources []model.Resource) []sysResp.ResourceMetaResp {
	var respArr []sysResp.ResourceMetaResp
	for _, res := range resources {
		if res.Pid == id {
			meta := make(map[string]any)
			meta["title"] = res.Title
			meta["icon"] = res.Icon
			meta["type"] = res.Type
			meta["hidden"] = res.Hidden
			meta["parentRoute"] = res.ParentRoute
			meta["order"] = res.Sort
			respArr = append(respArr, sysResp.ResourceMetaResp{
				Name:      res.Alias,
				Redirect:  res.Redirect,
				Path:      res.Path,
				Component: res.Component,
				Meta:      meta,
				Children:  getMenus(res.ID, resources),
			})
		}
	}
	return respArr
}

func (s *ResourceService) ListTree(params *sysReq.ResourceParams) (respArr []sysResp.ResourceTreeResp, err error) {
	var resources []model.Resource
	// 追加查询条件
	err = s.dbWhere(params.Title).Where("type IN (1,2,3)").Order("sort DESC").Find(&resources).Error
	if err == nil {
		respArr = getChildResource(snowflake.ParseInt64(1), resources)
	}
	return
}

// 递归子节点
func getChildResource(id snowflake.ID, resources []model.Resource) []sysResp.ResourceTreeResp {
	var respArr []sysResp.ResourceTreeResp
	for _, res := range resources {
		if res.Pid == id {
			respArr = append(respArr, sysResp.ResourceTreeResp{
				Resource: res,
				Children: getChildResource(res.ID, resources),
			})
		}
	}
	return respArr
}

func (s *ResourceService) dbWhere(title string) *gorm.DB {
	t := s.getDB()
	if len(title) > 0 {
		t.Where("title like ?", "%"+title+"%")
	}
	return t
}
