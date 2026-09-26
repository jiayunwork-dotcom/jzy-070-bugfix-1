package gravity

import "math"

// Observation 是单个测点的野外记录。
type Observation struct {
	// Gobs 测点绝对重力观测值，单位 m/s^2
	Gobs float64 `json:"gobs"`
	// H 测点高程，单位 m；可为负（低于参考面），但必须提供
	H float64 `json:"h"`
	// Phi 测点地理纬度，单位度，合理范围 [-90, 90]
	Phi float64 `json:"phi"`
	// Density 测点下方中间层平均密度，单位 g/cm^3，必须为正
	Density float64 `json:"rho"`
}

// PointResult 是单点归算的完整结果。
// 所有重力量同时以 mGal（内部统一计算单位）和 m/s^2 给出，
// 避免调用方把两个不同量纲的数悄悄相加。
type PointResult struct {
	Input             Observation `json:"input"`
	NormalGravity     Gravity     `json:"normal_gravity"`      // 理论正常重力 γ
	FreeAirCorrection Gravity     `json:"free_air_correction"` // 自由空气改正 Δg_FA（正）
	BouguerCorrection Gravity     `json:"bouguer_correction"`  // 布格板改正 Δg_B（异常中取负）
	TerrainCorrection Gravity     `json:"terrain_correction"`  // 地形改正，本服务从简，恒为 0
	BouguerAnomaly    Gravity     `json:"bouguer_anomaly"`     // 布格重力异常 Δg_Bouguer
}

// Gravity 同时承载一个重力量的两种单位表示。
type Gravity struct {
	MGal float64 `json:"mgal"`
	MS2  float64 `json:"m_s2"`
}

func inMGal(v float64) Gravity { return Gravity{MGal: v, MS2: MGalToMS2(v)} }

// ReducePoint 执行一个测点的完整归算。
//
// 物理链条（先统一到 mGal 再做加减）：
//
//	γ            = NormalGravity(φ)                       [m/s^2 → mGal]
//	Δg_FA        = 0.3086 · h                             [mGal]
//	Δg_B         = 0.04193 · ρ · h                        [mGal]
//	Δg_Bouguer   = gobs − γ + Δg_FA − Δg_B (+ Δg_T=0)     [mGal]
//
// 自由空气项为正、布格板项为负，二者符号相反、缺一不可。
func ReducePoint(obs Observation) PointResult {
	gammaMGal := MS2ToMGal(NormalGravity(obs.Phi))
	gobsMGal := MS2ToMGal(obs.Gobs)

	fa := FreeAirCorrection(obs.H)
	bc := BouguerCorrection(obs.Density, obs.H)
	terrain := 0.0

	anomaly := gobsMGal - gammaMGal + fa - bc + terrain

	return PointResult{
		Input:             obs,
		NormalGravity:     inMGal(gammaMGal),
		FreeAirCorrection: inMGal(fa),
		BouguerCorrection: inMGal(bc),
		TerrainCorrection: inMGal(terrain),
		BouguerAnomaly:    inMGal(anomaly),
	}
}

// ScanPoint 是高程扫描点列中的一个点：在其余参数固定时，
// 该高程对应的归算结果。
type ScanPoint struct {
	H                 float64 `json:"h"`
	NormalGravity     Gravity `json:"normal_gravity"`
	FreeAirCorrection Gravity `json:"free_air_correction"`
	BouguerCorrection Gravity `json:"bouguer_correction"`
	BouguerAnomaly    Gravity `json:"bouguer_anomaly"`
}

// ScanResult 是一次高程扫描的全部结果。
type ScanResult struct {
	Gobs    float64     `json:"gobs"`
	Phi     float64     `json:"phi"`
	Density float64     `json:"rho"`
	Points  []ScanPoint `json:"points"`
}

// ScanHeights 固定 gobs/φ/ρ，对给定高程采样序列逐点做真实归算。
// 每个点独立调用 ReducePoint，而不是由首尾两点插值出一条直线。
func ScanHeights(gobs, phi, density float64, heights []float64) ScanResult {
	points := make([]ScanPoint, 0, len(heights))
	for _, h := range heights {
		r := ReducePoint(Observation{Gobs: gobs, H: h, Phi: phi, Density: density})
		points = append(points, ScanPoint{
			H:                 h,
			NormalGravity:     r.NormalGravity,
			FreeAirCorrection: r.FreeAirCorrection,
			BouguerCorrection: r.BouguerCorrection,
			BouguerAnomaly:    r.BouguerAnomaly,
		})
	}
	return ScanResult{Gobs: gobs, Phi: phi, Density: density, Points: points}
}

// finite 判断一个浮点数是否为可参与物理计算的有限数
// （拒绝 NaN 与 ±Inf，JSON 解析出的非法数值会在这里被拦住）。
func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
