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

// normalGravityAnchors 是按国际正常重力公式标准式
//
//	γ(φ) = 9.780318·(1 + 5.3024e-3·sin²φ − 5.9e-6·sin²2φ)   [m/s²]
//
// 独立计算并换算到 mGal 的锚点值，0–90° 每 5° 一个，重点覆盖中纬度段。
// 历史上 sin²(2φ) 项符号被写反时，赤道与两极结果不变（sin2φ 在该两处为 0），
// 中纬度 γ 却被稳定抬高，45° 处达 ~11.5 mGal——下表对此逐点钉死。
var normalGravityAnchors = []struct {
	phi       float64 // 纬度，度
	gammaMGal float64 // 标准正常重力，mGal
}{
	{0, 978031.800000},
	{5, 978071.018858},
	{10, 978187.499489},
	{15, 978377.747892},
	{20, 978636.052726},
	{25, 978954.650490},
	{30, 979323.951163},
	{35, 979732.818692},
	{40, 980168.899103},
	{45, 980618.987521},
	{50, 981069.423935},
	{55, 981506.506363},
	{60, 981916.909072},
	{65, 982288.092922},
	{70, 982608.694720},
	{75, 982868.882731},
	{80, 983060.666313},
	{85, 983178.148961},
	{90, 983217.715816},
}

// 锚点值保留到 1e-6 mGal，容差取 1e-3 mGal：
// 比舍入噪声宽三个量级，又比曾出现的 11.5 mGal 偏差窄四个量级。
const anchorTolMGal = 1e-3

// 回归：任意纬度的正常重力都必须与国际正常重力公式标准值逐点对上，
// 中纬度（30°–60°）是历史上出错且赤道/两极测试照不到的地带。
func TestNormalGravity_MidLatitudeAnchorValues(t *testing.T) {
	for _, c := range normalGravityAnchors {
		got := MS2ToMGal(NormalGravity(c.phi))
		if d := got - c.gammaMGal; math.Abs(d) > anchorTolMGal {
			t.Fatalf("φ=%v° 正常重力应为 %.4f mGal，得到 %.4f mGal（偏差 %.4f mGal）",
				c.phi, c.gammaMGal, got, d)
		}
		// 南纬同纬度必须取同一标准值。
		gotS := MS2ToMGal(NormalGravity(-c.phi))
		if d := gotS - c.gammaMGal; math.Abs(d) > anchorTolMGal {
			t.Fatalf("φ=%v°（南纬）正常重力应为 %.4f mGal，得到 %.4f mGal", -c.phi, c.gammaMGal, gotS)
		}
	}
}

// 回归：sin²(2φ) 项必须把中纬度 γ 向下拉，而不是向上抬。
// 45° 处 sin²φ=0.5、sin²(2φ)=1 同时取到极值，对该项符号最敏感：
// γ(45°) 必须恰比 g_e(1+β1·sin²45°) 低 g_e·β2。
func TestNormalGravity_SecondHarmonicPullsDown(t *testing.T) {
	withoutSecondHarmonic := normalGravityEquator * (1 + beta1*0.5)
	want := withoutSecondHarmonic - normalGravityEquator*beta2
	got := NormalGravity(45)
	if !approxEq(got, want) {
		t.Fatalf("γ(45°) 应为 %v m/s^2（第二谐波项取负），得到 %v", want, got)
	}
	if got >= withoutSecondHarmonic {
		t.Fatalf("γ(45°)=%v 必须低于仅含 sin²φ 项的值 %v（sin²(2φ) 项符号疑被写反）",
			got, withoutSecondHarmonic)
	}
}

// 回归（全链路）：固定 gobs/h/ρ，让纬度从赤道扫到两极，
// 每个纬度的布格异常都必须等于 gobs − γ标准(φ) + Δg_FA − Δg_B。
// γ 在纬度上的任何偏差都会一比一漏进异常，本用例把整条链在
// 0–90° 范围（含最敏感的 45° 附近）钉在标准正常重力公式上。
func TestBouguerAnomaly_LatitudeSweepMatchesStandardGamma(t *testing.T) {
	base := Observation{Gobs: 9.802640, H: 500, Phi: 0, Density: 2.67}
	gobsMGal := MS2ToMGal(base.Gobs)
	fa := FreeAirCorrection(base.H)
	bc := BouguerCorrection(base.Density, base.H)

	for _, c := range normalGravityAnchors {
		obs := base
		obs.Phi = c.phi
		got := ReducePoint(obs).BouguerAnomaly.MGal
		want := gobsMGal - c.gammaMGal + fa - bc
		if d := got - want; math.Abs(d) > anchorTolMGal {
			t.Fatalf("φ=%v° 布格异常应为 %.4f mGal，得到 %.4f mGal（偏差 %.4f mGal）",
				c.phi, want, got, d)
		}
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
