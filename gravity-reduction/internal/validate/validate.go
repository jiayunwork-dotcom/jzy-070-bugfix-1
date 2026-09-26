// Package validate 集中处理重力归算服务的输入校验。
// 不合物理的输入一律拒绝并给出原因，不允许带着脏数据进入计算。
package validate

import (
	"errors"
	"math"

	"gravity-reduction/internal/gravity"
)

// 纬度的物理取值范围（度）。
const (
	MinLatitude = -90.0
	MaxLatitude = 90.0
)

// 合理的高程范围（m），仅用于拦截明显非物理的数值。
// 低于参考面（负高程，如盆地、井下）是允许的。
const (
	MinHeight = -12000.0
	MaxHeight = 10000.0
)

// FieldError 指明哪个字段、因为什么原因不合法。
type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// Error 实现 error 接口。
func (e FieldError) Error() string {
	return e.Field + ": " + e.Reason
}

// AsFieldError 从 error 中取出 FieldError。
func AsFieldError(err error) (FieldError, bool) {
	var fe FieldError
	return fe, errors.As(err, &fe)
}

func finite(field string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return FieldError{Field: field, Reason: "必须是有限数值，不能是 NaN 或无穷大"}
	}
	return nil
}

// Latitude 校验地理纬度（度）。
func Latitude(phi float64) error {
	if err := finite("phi", phi); err != nil {
		return err
	}
	if phi < MinLatitude || phi > MaxLatitude {
		return FieldError{
			Field:  "phi",
			Reason: "纬度超出合理范围 [-90, 90] 度",
		}
	}
	return nil
}

// Density 校验中间层平均密度（g/cm^3），必须为正的有限值。
func Density(rho float64) error {
	if err := finite("rho", rho); err != nil {
		return err
	}
	if rho <= 0 {
		return FieldError{
			Field:  "rho",
			Reason: "中间层密度必须为正值（g/cm^3）",
		}
	}
	return nil
}

// Height 校验单点高程（m）。
func Height(h float64) error {
	if err := finite("h", h); err != nil {
		return err
	}
	if h < MinHeight || h > MaxHeight {
		return FieldError{
			Field:  "h",
			Reason: "高程超出合理范围 [-12000, 10000] 米",
		}
	}
	return nil
}

// Gobs 校验绝对重力观测值（m/s^2），地表重力应为正的有限值。
func Gobs(g float64) error {
	if err := finite("gobs", g); err != nil {
		return err
	}
	if g <= 0 {
		return FieldError{
			Field:  "gobs",
			Reason: "绝对重力观测值必须为正值（m/s^2）",
		}
	}
	return nil
}

// Observation 校验一个测点的全部字段。
func Observation(o gravity.Observation) error {
	if err := Gobs(o.Gobs); err != nil {
		return err
	}
	if err := Height(o.H); err != nil {
		return err
	}
	if err := Latitude(o.Phi); err != nil {
		return err
	}
	if err := Density(o.Density); err != nil {
		return err
	}
	return nil
}

// Heights 校验高程扫描采样序列：必须非空，且每个高程都合法。
func Heights(hs []float64) error {
	if len(hs) == 0 {
		return FieldError{
			Field:  "heights",
			Reason: "高程采样序列不能为空",
		}
	}
	for i, h := range hs {
		if err := Height(h); err != nil {
			fe, _ := AsFieldError(err)
			return FieldError{
				Field:  "heights[" + itoa(i) + "]",
				Reason: fe.Reason,
			}
		}
	}
	return nil
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}
