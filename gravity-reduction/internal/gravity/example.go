package gravity

// ExampleObservation 是一份可供手工验算的内置示例测点：
// 中纬度、数百米高程，使两项改正都落在明显的 mGal 量级。
//
//	纬度 φ = 40°N
//	高程 h = 500 m
//	密度 ρ = 2.67 g/cm^3（地壳岩石常用中间层密度）
//	gobs   = 9.802640 m/s^2
//
// 手工核对（mGal）：
//
//	γ(40°) = 9.780318×(1+5.3024e-3·sin²40°−5.9e-6·sin²80°)
//	       ≈ 9.801689 m/s^2 ≈ 980168.90 mGal
//	Δg_FA  = 0.3086 × 500           = 154.30 mGal（正）
//	Δg_B   = 0.04193 × 2.67 × 500   ≈ 55.98 mGal（异常中取负）
//	Δg_Bouguer = 980264.00 − 980168.90 + 154.30 − 55.98 ≈ 193.42 mGal
var ExampleObservation = Observation{
	Gobs:    9.802640,
	H:       500,
	Phi:     40,
	Density: 2.67,
}

// ExampleHeights 是示例高程扫描使用的采样序列（m）。
var ExampleHeights = []float64{0, 100, 200, 300, 400, 500, 600, 800, 1000}
