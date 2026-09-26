package gravity

import "math"

// 国际正常重力公式（International Gravity Formula，GRS 1967）：
//
//	γ(φ) = g_e · (1 + β1·sin²φ − β2·sin²(2φ))
//
// 其中 φ 为测点地理纬度（度），结果以 m/s^2 计。
// 这是中国重力勘探教材中常用的"国际正常重力公式"形式，
// 赤道基准值与两个纬度系数如下。
//
// 注意：sin²(2φ) 项为减号。该项在赤道与两极恒为 0、在 45° 取最大值，
// 一旦符号写反，赤道/两极结果纹丝不动，中纬度 γ 却被抬高
// 2·g_e·β2 ≈ 11.5 mGal（45° 处最甚），并一比一漏进布格异常。
//
// 注意：纬度 φ 只在本步（计算正常重力）进入整条归算链，
// 自由空气改正与布格板改正都与纬度无关。
const (
	// normalGravityEquator 赤道正常重力 g_e，单位 m/s^2
	normalGravityEquator = 9.780318
	// beta1 对应 sin²φ 项的系数
	beta1 = 5.3024e-3
	// beta2 对应 sin²(2φ) 项的系数（该项在公式中取负号）
	beta2 = 5.9e-6
)

// NormalGravity 按纬度 phi（度）计算理论正常重力 γ，结果单位 m/s^2。
func NormalGravity(phiDeg float64) float64 {
	phi := phiDeg * math.Pi / 180.0
	sinPhi := math.Sin(phi)
	sin2Phi := math.Sin(2 * phi)
	return normalGravityEquator * (1 + beta1*sinPhi*sinPhi - beta2*sin2Phi*sin2Phi)
}
