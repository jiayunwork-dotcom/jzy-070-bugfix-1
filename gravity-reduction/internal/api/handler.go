// Package api 只负责 HTTP 编解码与路由，物理计算全部委托给
// gravity 包，输入合法性委托给 validate 包。
package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"gravity-reduction/internal/gravity"
	"gravity-reduction/internal/validate"
)

// 单次请求体大小上限 1 MiB，防止异常大请求体。
const maxBodyBytes = 1 << 20

// singleRequest 单点归算请求。
//
// 数值字段使用指针，以便区分"字段缺失"与"字段为 0"：
// 例如高程 h=0 是合法测点，但缺失 h 必须报错。
type singleRequest struct {
	Gobs    *float64 `json:"gobs"` // 绝对重力观测值，m/s^2
	H       *float64 `json:"h"`    // 测点高程，m，必填
	Phi     *float64 `json:"phi"`  // 地理纬度，度
	Density *float64 `json:"rho"`  // 中间层平均密度，g/cm^3
}

// scanRequest 高程扫描请求：固定 gobs/φ/ρ，扫描高程序列。
type scanRequest struct {
	Gobs    *float64  `json:"gobs"`
	Phi     *float64  `json:"phi"`
	Density *float64  `json:"rho"`
	Heights []float64 `json:"heights"` // 高程采样序列（m），必须非空
}

type errorBody struct {
	Error validate.FieldError `json:"error"`
}

// Register 在给定引擎上挂载全部路由。
func Register(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/reduce", handleReduce)
		v1.POST("/reduce/scan", handleScan)
		v1.GET("/example", handleExample)
	}
}

// decodeJSON 严格解析请求体：未知字段直接拒绝，避免拼写错误被静默吞掉。
func decodeJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		c.JSON(http.StatusBadRequest, errorBody{Error: validate.FieldError{
			Field:  "body",
			Reason: "请求体不是合法 JSON：" + err.Error(),
		}})
		return false
	}
	// 请求体只允许包含一个 JSON 对象，拒绝尾部多余数据。
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		c.JSON(http.StatusBadRequest, errorBody{Error: validate.FieldError{
			Field:  "body",
			Reason: "请求体在 JSON 对象之后还存在多余数据",
		}})
		return false
	}
	return true
}

func handleReduce(c *gin.Context) {
	var req singleRequest
	if !decodeJSON(c, &req) {
		return
	}

	obs, err := buildObservation(req.Gobs, req.H, req.Phi, req.Density)
	if err != nil {
		respondValidationError(c, err)
		return
	}
	if err := validate.Observation(obs); err != nil {
		respondValidationError(c, err)
		return
	}

	c.JSON(http.StatusOK, gravity.ReducePoint(obs))
}

func handleScan(c *gin.Context) {
	var req scanRequest
	if !decodeJSON(c, &req) {
		return
	}

	if req.Gobs == nil {
		badRequest(c, "gobs", "缺少绝对重力观测值 gobs（m/s^2）")
		return
	}
	if req.Phi == nil {
		badRequest(c, "phi", "缺少地理纬度 phi（度）")
		return
	}
	if req.Density == nil {
		badRequest(c, "rho", "缺少中间层密度 rho（g/cm^3）")
		return
	}

	if err := validate.Gobs(*req.Gobs); err != nil {
		respondValidationError(c, err)
		return
	}
	if err := validate.Latitude(*req.Phi); err != nil {
		respondValidationError(c, err)
		return
	}
	if err := validate.Density(*req.Density); err != nil {
		respondValidationError(c, err)
		return
	}
	if err := validate.Heights(req.Heights); err != nil {
		respondValidationError(c, err)
		return
	}

	c.JSON(http.StatusOK, gravity.ScanHeights(*req.Gobs, *req.Phi, *req.Density, req.Heights))
}

func buildObservation(gobs, h, phi, rho *float64) (gravity.Observation, error) {
	switch {
	case gobs == nil:
		return gravity.Observation{}, validate.FieldError{Field: "gobs", Reason: "缺少绝对重力观测值 gobs（m/s^2）"}
	case h == nil:
		return gravity.Observation{}, validate.FieldError{Field: "h", Reason: "缺少测点高程 h（米），高程为零时也必须显式给出"}
	case phi == nil:
		return gravity.Observation{}, validate.FieldError{Field: "phi", Reason: "缺少地理纬度 phi（度）"}
	case rho == nil:
		return gravity.Observation{}, validate.FieldError{Field: "rho", Reason: "缺少中间层密度 rho（g/cm^3）"}
	}
	return gravity.Observation{Gobs: *gobs, H: *h, Phi: *phi, Density: *rho}, nil
}

func badRequest(c *gin.Context, field, reason string) {
	c.JSON(http.StatusBadRequest, errorBody{
		Error: validate.FieldError{Field: field, Reason: reason},
	})
}

func respondValidationError(c *gin.Context, err error) {
	fe, ok := validate.AsFieldError(err)
	if !ok {
		fe = validate.FieldError{Field: "request", Reason: err.Error()}
	}
	c.JSON(http.StatusBadRequest, errorBody{Error: fe})
}
