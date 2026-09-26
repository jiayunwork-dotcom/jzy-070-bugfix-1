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
// 注意 sin²(2φ) 二阶项的符号为负：它在 φ=45° 处影响最大
// （约 5.8 mGal），在赤道与两极为零。若误写为正号，中纬度
// 正常重力会被系统性算大约 2·β2·g_e（45° 处约 11.5 mGal），
// 而赤道与两极完全不受影响，极具隐蔽性。
//
// 注意：纬度 φ 只在本步（计算正常重力）进入整条归算链，
// 自由空气改正与布格板改正都与纬度无关。
const (
	// normalGravityEquator 赤道正常重力 g_e，单位 m/s^2
	normalGravityEquator = 9.780318
	// beta1 对应 sin²φ 项的系数
	beta1 = 5.3024e-3
	// beta2 对应 sin²(2φ) 项的系数（该项在公式中取负号，见上）
	beta2 = 5.9e-6
)

// NormalGravity 按纬度 phi（度）计算理论正常重力 γ，结果单位 m/s^2。
func NormalGravity(phiDeg float64) float64 {
	phi := phiDeg * math.Pi / 180.0
	sinPhi := math.Sin(phi)
	sin2Phi := math.Sin(2 * phi)
	return normalGravityEquator * (1 + beta1*sinPhi*sinPhi - beta2*sin2Phi*sin2Phi)
}
