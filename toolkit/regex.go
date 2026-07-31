package toolkit

import (
	"regexp"
	"strconv"
)

func IsNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func IllegalNum(repo string) bool {
	// 仓库名在 1 到 100 个字符之间，字母、数字组成
	return !RegexpMatch(`^[a-zA-Z0-9]{1,100}$`, repo)
}

func IllegalSQLFieldName(fieldName string) bool {
	// 正则表达式匹配只包含字母、数字和下划线的字符串
	return !RegexpMatch(`^[a-zA-Z0-9_]{1,100}$`, fieldName)
}

func RegexpMatch(reg, s string) bool {
	return regexp.MustCompile(reg).MatchString(s)
}
