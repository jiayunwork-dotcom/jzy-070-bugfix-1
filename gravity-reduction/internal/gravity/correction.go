package gravity

// 两项高程相关改正，结果一律以 mGal 为单位。
//
// 自由空气改正（Free-air correction）：
//
//	Δg_FA = 0.3086 · h
//
// h 为测点高出参考面（大地水准面）的高程，单位 m；结果单位 mGal。
// 它把观测值从测点高程归算回参考面，量级随高程线性增大，符号为正。
const freeAirGradient = 0.3086 // mGal/m

// 布格板改正（Bouguer plate correction）：
//
//	Δg_B = 0.04193 · ρ · h
//
// h 单位 m，ρ 为测点与参考面之间中间层的平均密度，单位 g/cm^3；
// 结果单位 mGal。它扣除测点与参考面之间那层无限平板物质的引力贡献，
// 在布格异常中符号为负。
const bouguerCoefficient = 0.04193 // mGal/(g/cm^3)/m

// FreeAirCorrection 计算自由空气改正 Δg_FA（mGal）。
func FreeAirCorrection(h float64) float64 {
	return freeAirGradient * h
}

// BouguerCorrection 计算布格板改正 Δg_B（mGal），
// density 单位 g/cm^3，h 单位 m。
func BouguerCorrection(density, h float64) float64 {
	return bouguerCoefficient * density * h
}
