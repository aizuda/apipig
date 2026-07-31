package toolkit

import "math"

// CalculatePercentage 计算百分比
func CalculatePercentage(part, total int64) float64 {
	percentage := float64(part) / float64(total) * 100
	return math.Round(percentage*100) / 100
}

// HasDecimal 判断浮点数是否包含小数位
func HasDecimal(val float64) bool {
	return val != math.Trunc(val)
}
