package gravity

import (
	"math"
	"testing"
)

const tol = 1e-9

func approxEq(a, b float64) bool {
	return math.Abs(a-b) < tol
}

// 关系 1：高程为零时，自由空气改正与布格板改正同时归零，
// 此时布格异常恰等于 gobs 减去正常重力。
func TestZeroHeight_CorrectionsVanish(t *testing.T) {
	obs := Observation{Gobs: 9.802640, H: 0, Phi: 40, Density: 2.67}
	r := ReducePoint(obs)

	if r.FreeAirCorrection.MGal != 0 {
		t.Fatalf("h=0 时自由空气改正应为 0，得到 %v mGal", r.FreeAirCorrection.MGal)
	}
	if r.BouguerCorrection.MGal != 0 {
		t.Fatalf("h=0 时布格板改正应为 0，得到 %v mGal", r.BouguerCorrection.MGal)
	}

	want := MS2ToMGal(obs.Gobs) - MS2ToMGal(NormalGravity(obs.Phi))
	if !approxEq(r.BouguerAnomaly.MGal, want) {
		t.Fatalf("h=0 时布格异常应等于 gobs-γ：want %.6f, got %.6f", want, r.BouguerAnomaly.MGal)
	}
}

// 关系 2：其余参数不变，高程翻倍 → 自由空气与布格板改正都翻倍。
func TestDoubleHeight_BothCorrectionsDouble(t *testing.T) {
	base := Observation{Gobs: 9.8, H: 250, Phi: 30, Density: 2.67}
	doubled := base
	doubled.H = 500

	rb := ReducePoint(base)
	rd := ReducePoint(doubled)

	if !approxEq(rd.FreeAirCorrection.MGal, 2*rb.FreeAirCorrection.MGal) {
		t.Fatalf("高程翻倍后 FA 应翻倍：%v → %v", rb.FreeAirCorrection.MGal, rd.FreeAirCorrection.MGal)
	}
	if !approxEq(rd.BouguerCorrection.MGal, 2*rb.BouguerCorrection.MGal) {
		t.Fatalf("高程翻倍后布格板改正应翻倍：%v → %v", rb.BouguerCorrection.MGal, rd.BouguerCorrection.MGal)
	}
	// 正常重力与高程无关，必须保持不变。
	if rd.NormalGravity.MGal != rb.NormalGravity.MGal {
		t.Fatalf("正常重力不应随高程变化：%v → %v", rb.NormalGravity.MGal, rd.NormalGravity.MGal)
	}
}

// 关系 3：密度翻倍 → 只有布格板改正翻倍，自由空气改正毫不受影响。
func TestDoubleDensity_OnlyBouguerDoubles(t *testing.T) {
	base := Observation{Gobs: 9.8, H: 400, Phi: 45, Density: 2.67}
	doubled := base
	doubled.Density = 5.34

	rb := ReducePoint(base)
	rd := ReducePoint(doubled)

	if !approxEq(rd.BouguerCorrection.MGal, 2*rb.BouguerCorrection.MGal) {
		t.Fatalf("密度翻倍后布格板改正应翻倍：%v → %v", rb.BouguerCorrection.MGal, rd.BouguerCorrection.MGal)
	}
	if !approxEq(rd.FreeAirCorrection.MGal, rb.FreeAirCorrection.MGal) {
		t.Fatalf("自由空气改正不应受密度影响：%v → %v", rb.FreeAirCorrection.MGal, rd.FreeAirCorrection.MGal)
	}
	if rd.NormalGravity.MGal != rb.NormalGravity.MGal {
		t.Fatalf("正常重力不应受密度影响：%v → %v", rb.NormalGravity.MGal, rd.NormalGravity.MGal)
	}
}

// 关系 4：同一测点只改纬度 → 正常重力变，两项高程相关改正保持不变。
func TestLatitudeOnlyChangesNormalGravity(t *testing.T) {
	lo := Observation{Gobs: 9.8, H: 500, Phi: 10, Density: 2.67}
	hi := lo
	hi.Phi = 60

	rl := ReducePoint(lo)
	rh := ReducePoint(hi)

	if rh.NormalGravity.MGal <= rl.NormalGravity.MGal {
		t.Fatalf("正常重力应随纬度增大：φ=10° 得 %v，φ=60° 得 %v",
			rl.NormalGravity.MGal, rh.NormalGravity.MGal)
	}
	if rh.FreeAirCorrection.MGal != rl.FreeAirCorrection.MGal {
		t.Fatalf("自由空气改正不应随纬度变化：%v → %v",
			rl.FreeAirCorrection.MGal, rh.FreeAirCorrection.MGal)
	}
	if rh.BouguerCorrection.MGal != rl.BouguerCorrection.MGal {
		t.Fatalf("布格板改正不应随纬度变化：%v → %v",
			rl.BouguerCorrection.MGal, rh.BouguerCorrection.MGal)
	}
}

// 关键符号检查：自由空气改正对正高程必须为正、对负高程为负；
// 布格板改正恒为非负数值（在异常公式中以减号进入）。
// 一旦 FA 符号取反，高地测点异常会系统性漂移，本测试必须抓住。
func TestFreeAirSignDirection(t *testing.T) {
	above := ReducePoint(Observation{Gobs: 9.8, H: 1000, Phi: 45, Density: 2.67})
	if above.FreeAirCorrection.MGal <= 0 {
		t.Fatalf("正高程自由空气改正必须为正，得到 %v", above.FreeAirCorrection.MGal)
	}
	wantFA := 0.3086 * 1000
	if !approxEq(above.FreeAirCorrection.MGal, wantFA) {
		t.Fatalf("自由空气改正常量不符：want %v, got %v", wantFA, above.FreeAirCorrection.MGal)
	}
	if above.BouguerCorrection.MGal <= 0 {
		t.Fatalf("正高程正密度布格板改正数值必须为正，得到 %v", above.BouguerCorrection.MGal)
	}

	// 完整合成：用符号取反的错误实现算出的结果会与正确值相差 2·Δg_FA。
	below := ReducePoint(Observation{Gobs: 9.8, H: -100, Phi: 45, Density: 2.67})
	if below.FreeAirCorrection.MGal >= 0 {
		t.Fatalf("负高程自由空气改正必须为负，得到 %v", below.FreeAirCorrection.MGal)
	}

	// 直接验证合成公式 gobs − γ + FA − B，确保不是 +B 或 −FA。
	r := above
	gobsMGal := MS2ToMGal(r.Input.Gobs)
	want := gobsMGal - r.NormalGravity.MGal + r.FreeAirCorrection.MGal - r.BouguerCorrection.MGal
	if !approxEq(r.BouguerAnomaly.MGal, want) {
		t.Fatalf("布格异常合成符号错误：want %.6f, got %.6f", want, r.BouguerAnomaly.MGal)
	}
}

// 单位链：mGal 与 m/s^2 必须严格互逆，且每个输出量的两个单位表示一致。
func TestUnitConversion(t *testing.T) {
	if MS2ToMGal(1.0) != 1e5 {
		t.Fatalf("1 m/s^2 应为 1e5 mGal，得到 %v", MS2ToMGal(1.0))
	}
	if !approxEq(MGalToMS2(MS2ToMGal(9.81)), 9.81) {
		t.Fatal("mGal/m,s^2 换算不互逆")
	}
	r := ReducePoint(Observation{Gobs: 9.802640, H: 500, Phi: 40, Density: 2.67})
	for name, g := range map[string]Gravity{
		"normal":  r.NormalGravity,
		"fa":      r.FreeAirCorrection,
		"bouguer": r.BouguerCorrection,
		"anomaly": r.BouguerAnomaly,
	} {
		if !approxEq(MS2ToMGal(g.MS2), g.MGal) {
			t.Fatalf("%s 的两种单位表示不一致：%v mGal vs %v m/s^2", name, g.MGal, g.MS2)
		}
	}
}

// 正常重力：两极/赤道/中纬度的已知量级与公式系数核对。
func TestNormalGravityValues(t *testing.T) {
	eq := NormalGravity(0)
	pole := NormalGravity(90)
	mid := NormalGravity(45)

	if !approxEq(eq, normalGravityEquator) {
		t.Fatalf("赤道正常重力应为 %v，得到 %v", normalGravityEquator, eq)
	}
	wantPole := normalGravityEquator * (1 + beta1)
	if !approxEq(pole, wantPole) {
		t.Fatalf("两极正常重力应为 %v，得到 %v", wantPole, pole)
	}
	if !(eq < mid && mid < pole) {
		t.Fatalf("正常重力应满足赤道 < 中纬 < 两极：%v < %v < %v", eq, mid, pole)
	}
	// 北纬/南纬同纬度正常重力相同（公式只含 sin 的偶次项）。
	if !approxEq(NormalGravity(35), NormalGravity(-35)) {
		t.Fatal("北纬与南纬同纬度的正常重力应相等")
	}
	// 40° 的绝对量级 sanity check：落在 [9.78, 9.84] m/s^2。
	g40 := NormalGravity(40)
	if g40 < 9.78 || g40 > 9.84 {
		t.Fatalf("40° 正常重力 %v 超出合理地表量级", g40)
	}
}

// 回归：中纬度正常重力必须逐点贴住国际正常重力公式（GRS 1967）标准值。
// 参考值由公式 γ=g_e·(1+β1·sin²φ−β2·sin²2φ) 独立计算（g_e=9.780318，
// β1=5.3024e-3，β2=5.9e-6）。sin²(2φ) 项若被误写为正号，赤道与两极
// 不受影响，但中纬度会被算大约 2·β2·sin²(2φ)·g_e，45° 处达 11.5 mGal，
// 本表在偏差最大的一带逐点卡住。
func TestNormalGravity_MidLatitudeReferenceValues(t *testing.T) {
	cases := []struct {
		phi  float64
		want float64 // m/s^2
	}{
		{15, 9.783777478918},
		{30, 9.793239511634},
		{40, 9.801688991035},
		{45, 9.806189875205}, // sin²(2φ) 项影响最大的纬度
		{50, 9.810694239345},
		{60, 9.819169090715},
		{75, 9.828688827307},
		{-45, 9.806189875205}, // 南纬中纬度同样卡住
	}
	for _, tc := range cases {
		if got := NormalGravity(tc.phi); !approxEq(got, tc.want) {
			t.Fatalf("φ=%v° 正常重力不符国际正常重力公式：want %.12f, got %.12f（差 %.4f mGal）",
				tc.phi, tc.want, got, (got-tc.want)*1e5)
		}
	}
}

// 回归：全纬度扫描（-90°..90°，步长 0.5°），与用恒等式
// sin²(2φ)=4·sin²φ·cos²φ 独立写出的参考式逐点比对。
// 任何只在中纬度冒头的纬度相关偏差（符号、系数、角度单位）
// 都会被这条曲线卡住，而赤道/两极的正确值不受影响。
func TestNormalGravity_FullLatitudeSweepMatchesIGF(t *testing.T) {
	reference := func(phiDeg float64) float64 {
		phi := phiDeg * math.Pi / 180.0
		s2 := math.Sin(phi)
		s2 = s2 * s2 // sin²φ
		sin2phiSq := 4 * s2 * (1 - s2)
		return normalGravityEquator * (1 + beta1*s2 - beta2*sin2phiSq)
	}
	for phi := -90.0; phi <= 90.0001; phi += 0.5 {
		if got, want := NormalGravity(phi), reference(phi); !approxEq(got, want) {
			t.Fatalf("φ=%.1f° 正常重力偏离国际正常重力公式：want %.12f, got %.12f（差 %.4f mGal）",
				phi, want, got, (got-want)*1e5)
		}
	}
}

// 回归（端到端）：中纬度测点的布格异常手算核对。
// φ=45°、h=500 m、ρ=2.67 g/cm³、gobs=9.806190 m/s²：
//
//	γ(45°) = 9.780318×(1+5.3024e-3×0.5−5.9e-6×1) = 980618.9875205 mGal
//	Δg_FA  = 0.3086×500          = 154.30 mGal
//	Δg_B   = 0.04193×2.67×500    = 55.97655 mGal
//	Δg_Bouguer = 980619.0 − 980618.9875205 + 154.30 − 55.97655
//	           = 98.3359295 mGal
//
// 正常重力若在 45° 被算大 11.5 mGal（sin²(2φ) 项符号错误），
// 会一比一漏进布格异常，本用例在最终输出上把它卡住。
func TestBouguerAnomaly_MidLatitudeHandCheck(t *testing.T) {
	r := ReducePoint(Observation{Gobs: 9.806190, H: 500, Phi: 45, Density: 2.67})

	const wantGamma = 980618.9875205  // mGal
	const wantAnomaly = 98.3359295    // mGal
	const mGalTol = 1e-6              // 远小于 11.5 mGal 级回归，远大于浮点噪声

	if d := math.Abs(r.NormalGravity.MGal - wantGamma); d > mGalTol {
		t.Fatalf("45° 正常重力不符：want %.7f mGal, got %.7f mGal", wantGamma, r.NormalGravity.MGal)
	}
	if d := math.Abs(r.BouguerAnomaly.MGal - wantAnomaly); d > mGalTol {
		t.Fatalf("45° 布格异常不符：want %.7f mGal, got %.7f mGal", wantAnomaly, r.BouguerAnomaly.MGal)
	}
}

// 高程扫描：每个点都必须由真实公式独立算出，
// 且 γ 不变、FA 与 B 随 h 线性变化，不允许退化成写死的直线/常数。
func TestScanHeights_PointwiseFormula(t *testing.T) {
	heights := []float64{0, 123, 311, 500, 742, 1000}
	res := ScanHeights(9.802640, 40, 2.67, heights)

	if len(res.Points) != len(heights) {
		t.Fatalf("扫描点数应等于采样数 %d，得到 %d", len(heights), len(res.Points))
	}

	gamma0 := res.Points[0].NormalGravity.MGal
	for i, p := range res.Points {
		h := heights[i]
		single := ReducePoint(Observation{Gobs: 9.802640, H: h, Phi: 40, Density: 2.67})

		if p.H != h {
			t.Fatalf("第 %d 点高程错位：want %v, got %v", i, h, p.H)
		}
		if !approxEq(p.FreeAirCorrection.MGal, single.FreeAirCorrection.MGal) {
			t.Fatalf("第 %d 点 FA 与逐点公式结果不符", i)
		}
		if !approxEq(p.BouguerCorrection.MGal, single.BouguerCorrection.MGal) {
			t.Fatalf("第 %d 点布格板改正与逐点公式结果不符", i)
		}
		if !approxEq(p.BouguerAnomaly.MGal, single.BouguerAnomaly.MGal) {
			t.Fatalf("第 %d 点异常与逐点公式结果不符", i)
		}
		if !approxEq(p.NormalGravity.MGal, gamma0) {
			t.Fatalf("扫描中正常重力不应随高程变化（第 %d 点）", i)
		}
	}

	// 扫描点列不得退化为常数直线（相邻点必须有真实差异）。
	for i := 1; i < len(res.Points); i++ {
		if res.Points[i].FreeAirCorrection.MGal == res.Points[i-1].FreeAirCorrection.MGal {
			t.Fatalf("FA 扫描点列在第 %d 点退化", i)
		}
	}
}

// 内置示例数值核对：两项改正落在明显的 mGal 量级。
func TestExampleSanity(t *testing.T) {
	r := ReducePoint(ExampleObservation)
	if r.FreeAirCorrection.MGal < 100 || r.FreeAirCorrection.MGal > 200 {
		t.Fatalf("示例 FA 量级异常：%v", r.FreeAirCorrection.MGal)
	}
	if r.BouguerCorrection.MGal < 40 || r.BouguerCorrection.MGal > 70 {
		t.Fatalf("示例布格板改正量级异常：%v", r.BouguerCorrection.MGal)
	}
}
