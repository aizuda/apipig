package service

import (
	"errors"
	"fmt"

	aiReq "apipig/app/ai/model/request"
)

func validateStatusChange(params *aiReq.StatusChangeParams) error {
	if params == nil {
		return errors.New("状态参数不能为空")
	}
	if params.ID <= 0 {
		return errors.New("资源 ID 不能为空")
	}
	if params.Status != gatewayStatusNormal && params.Status != gatewayStatusDisabled {
		return errors.New("状态仅支持启用或禁用")
	}
	return nil
}

func changeResourceStatus(store AIStore, resource any, resourceName string, params *aiReq.StatusChangeParams) (bool, error) {
	if err := validateStatusChange(params); err != nil {
		return false, err
	}
	result := store.Query(resource).Where("id = ?", params.ID).Update("status", params.Status)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, fmt.Errorf("%s不存在", resourceName)
	}
	return true, nil
}
