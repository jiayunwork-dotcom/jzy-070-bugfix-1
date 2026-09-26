package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"gravity-reduction/internal/gravity"
)

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r)
	return r
}

func postJSON(t *testing.T, r *gin.Engine, path string, body any) (int, map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("编码请求失败：%v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var out map[string]any
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("响应不是 JSON（%d）：%s", w.Code, w.Body.String())
		}
	}
	return w.Code, out
}

func validBody() map[string]any {
	return map[string]any{
		"gobs": 9.802640,
		"h":    500.0,
		"phi":  40.0,
		"rho":  2.67,
	}
}

func TestReduce_SuccessMatchesPhysics(t *testing.T) {
	r := newRouter()
	code, out := postJSON(t, r, "/api/v1/reduce", validBody())
	if code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d：%v", code, out)
	}

	want := gravity.ReducePoint(gravity.ExampleObservation)

	fa := out["free_air_correction"].(map[string]any)["mgal"].(float64)
	bc := out["bouguer_correction"].(map[string]any)["mgal"].(float64)
	ba := out["bouguer_anomaly"].(map[string]any)["mgal"].(float64)
	ng := out["normal_gravity"].(map[string]any)["mgal"].(float64)

	if d := fa - want.FreeAirCorrection.MGal; d > 1e-6 || d < -1e-6 {
		t.Fatalf("FA 不符：want %v, got %v", want.FreeAirCorrection.MGal, fa)
	}
	if d := bc - want.BouguerCorrection.MGal; d > 1e-6 || d < -1e-6 {
		t.Fatalf("布格板改正不符：want %v, got %v", want.BouguerCorrection.MGal, bc)
	}
	if d := ba - want.BouguerAnomaly.MGal; d > 1e-6 || d < -1e-6 {
		t.Fatalf("布格异常不符：want %v, got %v", want.BouguerAnomaly.MGal, ba)
	}
	if d := ng - want.NormalGravity.MGal; d > 1e-3 || d < -1e-3 {
		t.Fatalf("正常重力不符：want %v, got %v", want.NormalGravity.MGal, ng)
	}
}

// 高程为零也要能正常处理（验证"缺失"与"零值"被正确区分）。
func TestReduce_ZeroHeightAccepted(t *testing.T) {
	r := newRouter()
	body := validBody()
	body["h"] = 0.0
	code, out := postJSON(t, r, "/api/v1/reduce", body)
	if code != http.StatusOK {
		t.Fatalf("h=0 应返回 200，得到 %d：%v", code, out)
	}
	fa := out["free_air_correction"].(map[string]any)["mgal"].(float64)
	bc := out["bouguer_correction"].(map[string]any)["mgal"].(float64)
	if fa != 0 || bc != 0 {
		t.Fatalf("h=0 时两项改正应为 0，得到 FA=%v B=%v", fa, bc)
	}
}

func TestReduce_RejectsInvalidInputs(t *testing.T) {
	r := newRouter()

	cases := []struct {
		name   string
		mutate func(map[string]any)
		field  string
	}{
		{"缺高程", func(b map[string]any) { delete(b, "h") }, "h"},
		{"缺重力", func(b map[string]any) { delete(b, "gobs") }, "gobs"},
		{"缺纬度", func(b map[string]any) { delete(b, "phi") }, "phi"},
		{"缺密度", func(b map[string]any) { delete(b, "rho") }, "rho"},
		{"纬度超界", func(b map[string]any) { b["phi"] = 91.0 }, "phi"},
		{"密度为零", func(b map[string]any) { b["rho"] = 0.0 }, "rho"},
		{"密度为负", func(b map[string]any) { b["rho"] = -2.0 }, "rho"},
		{"观测值为负", func(b map[string]any) { b["gobs"] = -9.8 }, "gobs"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := validBody()
			tc.mutate(body)
			code, out := postJSON(t, r, "/api/v1/reduce", body)
			if code != http.StatusBadRequest {
				t.Fatalf("期望 400，得到 %d：%v", code, out)
			}
			errObj := out["error"].(map[string]any)
			if errObj["field"] != tc.field {
				t.Fatalf("错误字段应为 %s，得到 %v", tc.field, errObj["field"])
			}
			if errObj["reason"] == "" {
				t.Fatal("错误必须带原因")
			}
		})
	}
}

func TestReduce_RejectsMalformedJSON(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reduce", bytes.NewBufferString("{not json"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 JSON 应返回 400，得到 %d", w.Code)
	}
}

func TestReduce_RejectsUnknownField(t *testing.T) {
	r := newRouter()
	body := validBody()
	body["elevation_typo"] = 1
	code, _ := postJSON(t, r, "/api/v1/reduce", body)
	if code != http.StatusBadRequest {
		t.Fatalf("未知字段应返回 400，得到 %d", code)
	}
}

func TestScan_PointwiseResults(t *testing.T) {
	r := newRouter()
	body := map[string]any{
		"gobs":    9.802640,
		"phi":     40.0,
		"rho":     2.67,
		"heights": []float64{0, 250, 500, 1000},
	}
	code, out := postJSON(t, r, "/api/v1/reduce/scan", body)
	if code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d：%v", code, out)
	}
	points := out["points"].([]any)
	if len(points) != 4 {
		t.Fatalf("应返回 4 个点，得到 %d", len(points))
	}
	for i, p := range points {
		pm := p.(map[string]any)
		h := body["heights"].([]float64)[i]
		if pm["h"].(float64) != h {
			t.Fatalf("第 %d 点高程错位", i)
		}
		want := gravity.ReducePoint(gravity.Observation{
			Gobs: 9.802640, H: h, Phi: 40, Density: 2.67,
		})
		fa := pm["free_air_correction"].(map[string]any)["mgal"].(float64)
		ba := pm["bouguer_anomaly"].(map[string]any)["mgal"].(float64)
		if fa != want.FreeAirCorrection.MGal {
			t.Fatalf("第 %d 点 FA 不符：%v vs %v", i, fa, want.FreeAirCorrection.MGal)
		}
		if ba != want.BouguerAnomaly.MGal {
			t.Fatalf("第 %d 点异常不符：%v vs %v", i, ba, want.BouguerAnomaly.MGal)
		}
	}
}

func TestScan_RejectsEmptyAndBadInputs(t *testing.T) {
	r := newRouter()

	t.Run("空序列", func(t *testing.T) {
		body := map[string]any{
			"gobs": 9.8, "phi": 40.0, "rho": 2.67, "heights": []float64{},
		}
		code, out := postJSON(t, r, "/api/v1/reduce/scan", body)
		if code != http.StatusBadRequest {
			t.Fatalf("空序列应 400，得到 %d：%v", code, out)
		}
	})

	t.Run("缺序列字段", func(t *testing.T) {
		body := map[string]any{"gobs": 9.8, "phi": 40.0, "rho": 2.67}
		code, _ := postJSON(t, r, "/api/v1/reduce/scan", body)
		if code != http.StatusBadRequest {
			t.Fatalf("缺 heights 应 400，得到 %d", code)
		}
	})

	t.Run("纬度超界", func(t *testing.T) {
		body := map[string]any{
			"gobs": 9.8, "phi": -91.0, "rho": 2.67, "heights": []float64{1, 2},
		}
		code, out := postJSON(t, r, "/api/v1/reduce/scan", body)
		if code != http.StatusBadRequest {
			t.Fatalf("非法纬度应 400，得到 %d：%v", code, out)
		}
	})
}

func TestExampleEndpoint(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/example", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("示例接口应 200，得到 %d", w.Code)
	}
}

func TestHealth(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("健康检查应 200，得到 %d", w.Code)
	}
}
