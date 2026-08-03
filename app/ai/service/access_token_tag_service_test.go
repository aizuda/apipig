package service

import (
	"testing"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	coreAPI "apipig/core/api"
	coreReq "apipig/core/api/request"
	"apipig/middleware"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAccessTokenTagLifecycleAndSort(t *testing.T) {
	database, err := gorm.Open(
		sqlite.Open("file:access-token-tag-lifecycle?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.AccessToken{},
		&model.AccessTokenTag{},
		&model.AccessTokenTagRelation{},
	))

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)
	ctx.Locals("tokenClaims", &middleware.TokenClaims{ID: snowflake.ID(1), Username: "tester"})

	tagService := &AccessTokenTagService{store: newGormAIStore(func() *gorm.DB { return database })}
	for _, tag := range []*model.AccessTokenTag{
		{Name: "生产环境", Remark: "线上业务"},
		{Name: "测试环境", Remark: "联调使用"},
	} {
		success, saveErr := tagService.Save(&aiReq.AccessTokenTagSaveParams{Ctx: ctx, Tag: tag})
		require.NoError(t, saveErr)
		assert.True(t, success)
	}

	_, err = tagService.Save(&aiReq.AccessTokenTagSaveParams{
		Ctx: ctx,
		Tag: &model.AccessTokenTag{Name: "生产环境"},
	})
	require.EqualError(t, err, "标签名称已存在")

	tags, err := tagService.List(&coreReq.Empty{})
	require.NoError(t, err)
	require.Len(t, tags, 2)
	assert.Equal(t, "生产环境", tags[0].Name)
	assert.Equal(t, 1, tags[0].Sort)
	assert.Equal(t, "测试环境", tags[1].Name)
	assert.Equal(t, 2, tags[1].Sort)

	first := tags[0]
	first.Remark = "核心线上业务"
	success, err := tagService.Save(&aiReq.AccessTokenTagSaveParams{Ctx: ctx, Tag: &first})
	require.NoError(t, err)
	assert.True(t, success)

	success, err = tagService.Sort(&aiReq.AccessTokenTagSortParams{IDs: []snowflake.ID{tags[1].ID, tags[0].ID}})
	require.NoError(t, err)
	assert.True(t, success)
	tags, err = tagService.List(&coreReq.Empty{})
	require.NoError(t, err)
	assert.Equal(t, "测试环境", tags[0].Name)
	assert.Equal(t, "核心线上业务", tags[1].Remark)

	token := model.AccessToken{
		MODEL: coreAPI.MODEL{ID: snowflake.ID(3601)},
		Name:  "tag cascade target",
		Token: "sk-tag-cascade",
		RPM:   60,
	}
	require.NoError(t, database.Create(&token).Error)
	require.NoError(t, database.Create(&[]model.AccessTokenTagRelation{
		{AccessTokenID: token.ID, TagID: tags[0].ID},
		{AccessTokenID: token.ID, TagID: tags[1].ID},
	}).Error)

	deletedTagID := tags[0].ID
	success, err = tagService.Delete(&coreReq.IdsReq{Ids: []snowflake.ID{deletedTagID}})
	require.NoError(t, err)
	assert.True(t, success)
	tags, err = tagService.List(&coreReq.Empty{})
	require.NoError(t, err)
	require.Len(t, tags, 1)
	var deletedTagCount int64
	require.NoError(t, database.Unscoped().Model(&model.AccessTokenTag{}).
		Where("id = ?", deletedTagID).
		Count(&deletedTagCount).Error)
	assert.Zero(t, deletedTagCount)
	var relationCount int64
	require.NoError(t, database.Model(&model.AccessTokenTagRelation{}).
		Where("access_token_id = ?", token.ID).
		Count(&relationCount).Error)
	assert.EqualValues(t, 1, relationCount)
	var tokenCount int64
	require.NoError(t, database.Model(&model.AccessToken{}).
		Where("id = ?", token.ID).
		Count(&tokenCount).Error)
	assert.EqualValues(t, 1, tokenCount)
	assert.Equal(t, "生产环境", tags[0].Name)
}
