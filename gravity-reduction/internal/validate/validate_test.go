package validate

import (
	"math"
	"testing"

	"gravity-reduction/internal/gravity"
)

func TestLatitudeValidation(t *testing.T) {
	for _, phi := range []float64{-90, 0, 45, 90, -45.5} {
		if err := Latitude(phi); err != nil {
			t.Fatalf("纬度 %v 应合法，得到错误 %v", phi, err)
		}
	}
	for _, phi := range []float64{-90.0001, 90.0001, 100, -100} {
		if err := Latitude(phi); err == nil {
			t.Fatalf("纬度 %v 应被拒绝", phi)
		}
	}
	if err := Latitude(math.NaN()); err == nil {
		t.Fatal("NaN 纬度应被拒绝")
	}
}

func TestDensityValidation(t *testing.T) {
	for _, rho := range []float64{0.001, 2.67, 8.9} {
		if err := Density(rho); err != nil {
			t.Fatalf("密度 %v 应合法，得到错误 %v", rho, err)
		}
	}
	for _, rho := range []float64{0, -1, -2.67} {
		fe := Density(rho)
		if fe == nil {
			t.Fatalf("密度 %v 应被拒绝", rho)
		}
	}
	if err := Density(math.Inf(1)); err == nil {
		t.Fatal("Inf 密度应被拒绝")
	}
}

func TestHeightValidation(t *testing.T) {
	if err := Height(0); err != nil {
		t.Fatal("高程 0 应合法")
	}
	if err := Height(-100); err != nil {
		t.Fatal("负高程（低于参考面）应合法")
	}
	if err := Height(math.NaN()); err == nil {
		t.Fatal("NaN 高程应被拒绝")
	}
	if err := Height(20000); err == nil {
		t.Fatal("超界高程应被拒绝")
	}
}

func TestGobsValidation(t *testing.T) {
	if err := Gobs(9.81); err != nil {
		t.Fatal("9.81 m/s^2 应合法")
	}
	if err := Gobs(0); err == nil {
		t.Fatal("零观测值应被拒绝")
	}
	if err := Gobs(-9.8); err == nil {
		t.Fatal("负观测值应被拒绝")
	}
}

func TestObservationValidation(t *testing.T) {
	ok := gravity.Observation{Gobs: 9.80264, H: 500, Phi: 40, Density: 2.67}
	if err := Observation(ok); err != nil {
		t.Fatalf("合法测点被拒：%v", err)
	}
	bad := ok
	bad.Phi = 91
	if err := Observation(bad); err == nil {
		t.Fatal("非法纬度测点应被拒绝")
	}
}

func TestHeightsValidation(t *testing.T) {
	if err := Heights(nil); err == nil {
		t.Fatal("nil 采样序列应被拒绝（序列为空）")
	}
	if err := Heights([]float64{}); err == nil {
		t.Fatal("空采样序列应被拒绝")
	}
	if err := Heights([]float64{0, 100, 200}); err != nil {
		t.Fatalf("合法序列被拒：%v", err)
	}
	err := Heights([]float64{0, 99999})
	if err == nil {
		t.Fatal("含非法高程的序列应被拒绝")
	}
	if fe, ok := AsFieldError(err); !ok || fe.Field != "heights[1]" {
		t.Fatalf("错误应定位到 heights[1]，得到 %v", err)
	}
}

func TestFieldErrorCarriesReason(t *testing.T) {
	err := Latitude(91)
	fe, ok := AsFieldError(err)
	if !ok {
		t.Fatal("错误类型应为 FieldError")
	}
	if fe.Field != "phi" || fe.Reason == "" {
		t.Fatalf("错误应带字段名与原因，得到 %+v", fe)
	}
}
