// Package gravity 实现重力勘探内业归算的全部物理计算。
//
// 子文件按职责拆分：
//   - units.go    mGal 与 m/s^2 之间的单位换算（唯一的换算入口）
//   - normal.go   理论正常重力 γ（国际正常重力公式，GRS 1967）
//   - correction.go 自由空气改正与布格板改正
//   - reduction.go 单点归算与高程扫描（把上面三者合成为布格重力异常）
package gravity

// mGalPerMS2 是 m/s^2 与 mGal 之间的换算系数。
//
//	1 Gal = 1 cm/s^2 = 1e-2 m/s^2
//	1 mGal = 1e-3 Gal = 1e-5 m/s^2
//	1 m/s^2 = 1e5 mGal
const mGalPerMS2 = 1e5

// MS2ToMGal 把以 m/s^2 计的重力值换算为 mGal。
func MS2ToMGal(v float64) float64 {
	return v * mGalPerMS2
}

// MGalToMS2 把以 mGal 计的重力值换算为 m/s^2。
func MGalToMS2(v float64) float64 {
	return v / mGalPerMS2
}
