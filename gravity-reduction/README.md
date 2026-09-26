# 重力勘探内业归算服务（gravity-reduction）

把野外测得的绝对重力观测值归算为**布格重力异常**的无状态 HTTP 后端服务。
仅提供 HTTP/JSON 接口，不含任何界面、用户系统，也不涉及导航定位、卫星
几何精度因子（GDOP）、测绘派工或轨迹记录等功能。

技术栈：Go 1.22 + [Gin](https://github.com/gin-gonic/gin)，静态二进制，
可一次构建为容器镜像。

---

## 1. 物理链条与符号约定

一个测点的输入为：

| 符号 | 含义 | 单位 |
| --- | --- | --- |
| gobs | 绝对重力观测值 | m/s² |
| h | 测点高程（相对参考面/大地水准面，可为负） | m |
| φ | 测点地理纬度 | 度（°） |
| ρ | 测点下方中间层平均密度 | g/cm³ |

归算分四步：

### 1.1 理论正常重力 γ（仅本步使用纬度）

国际正常重力公式（GRS 1967）：

```
γ(φ) = 9.780318 · (1 + 5.3024e-3·sin²φ − 5.9e-6·sin²2φ)     [m/s²]
```

纬度 φ **只**在这一步进入整条计算链，两项高程改正与纬度无关。
正常重力随纬度单调增大（赤道 9.780318 m/s²，两极约 9.832177 m/s²）。

### 1.2 自由空气改正 Δg_FA（符号：正）

把观测值从测点高程归算回参考面，随高程线性增大：

```
Δg_FA = 0.3086 · h        （h 以 m 计，结果 mGal）
```

### 1.3 布格板改正 Δg_B（在异常中符号：负）

扣除测点与参考面之间无限平板状中间层物质的引力贡献：

```
Δg_B = 0.04193 · ρ · h    （ρ 以 g/cm³ 计，h 以 m 计，结果 mGal）
```

### 1.4 合成布格重力异常

符号关系钉死为：

```
Δg_Bouguer = gobs − γ + Δg_FA − Δg_B (+ Δg_T)
```

- 自由空气项为 **正**、布格板项为 **负**，二者符号相反、缺一不可；
- 地形改正 Δg_T 本服务从简，缺省恒取 **0**（响应中仍显式返回该字段）。

**自由空气改正的符号是高危点**：一旦取反，所有高地测点的异常都会产生
大小为 2·Δg_FA 的系统性漂移，自动化测试中对此有专门断言。

---

## 2. 单位链（务必一致）

```
1 Gal   = 1 cm/s² = 1e-2 m/s²
1 mGal  = 1e-3 Gal = 1e-5 m/s²
1 m/s²  = 1e5 mGal
```

- 输入 gobs 与 γ 的公式原生单位是 **m/s²**；
- 两项改正的系数（0.3086、0.04193）原生给出的是 **mGal**；
- 内部合成前，gobs 与 γ 统一乘以 1e5 换算到 **mGal** 后再做加减，
  绝不允许两个不同量纲的数直接相加；
- 响应中的每个重力量都同时给出 `mgal` 与 `m_s2` 两种表示，
  换算只经过 `internal/gravity/units.go` 这一个入口。

---

## 3. 内置示例（可手工验算）

中纬度、数百米高程，改正量落在明显的 mGal 量级：

```
φ = 40°N,  h = 500 m,  ρ = 2.67 g/cm³,  gobs = 9.802640 m/s²

γ(40°) = 9.780318×(1+5.3024e-3·sin²40°−5.9e-6·sin²80°)
       ≈ 9.801689 m/s² ≈ 980168.90 mGal
Δg_FA  = 0.3086 × 500         = 154.30 mGal（正）
Δg_B   = 0.04193 × 2.67 × 500 ≈ 55.98 mGal（异常中取负）
Δg_Bouguer = 980264.00 − 980168.90 + 154.30 − 55.98 ≈ 193.42 mGal
```

服务启动后 `GET /api/v1/example` 可直接取回该示例的完整结果用于核对。

---

## 4. HTTP 接口

服务监听 `:8080`（可用环境变量 `PORT` 覆盖），所有接口均返回 JSON。

### 4.1 健康检查

```
GET /health
→ 200 {"status":"ok"}
```

### 4.2 单点归算

```
POST /api/v1/reduce
Content-Type: application/json

{
  "gobs": 9.802640,   // m/s²，必填，必须为正
  "h":    500,        // m，必填（h=0 也必须显式给出）
  "phi":  40,         // 度，必填，范围 [-90, 90]
  "rho":  2.67        // g/cm³，必填，必须 > 0
}
```

返回 γ、自由空气改正、布格板改正、地形改正（0）与布格异常四个量，
每个量同时带 `mgal` / `m_s2`：

```
{
  "input": { ... },
  "normal_gravity":     {"mgal": 980168.90, "m_s2": 9.80168899},
  "free_air_correction":{"mgal": 154.30,    "m_s2": 0.001543},
  "bouguer_correction": {"mgal": 55.97655,  "m_s2": 0.00055977},
  "terrain_correction": {"mgal": 0, "m_s2": 0},
  "bouguer_anomaly":    {"mgal": 193.42435, "m_s2": 0.00193424}
}
```

### 4.3 高程扫描

固定 gobs、φ、ρ，给定高程采样序列，逐点返回布格异常随高程的变化点列。
每个点都独立走完整真实公式，**不是**首尾插值或写死的直线：

```
POST /api/v1/reduce/scan
{
  "gobs": 9.802640,
  "phi":  40,
  "rho":  2.67,
  "heights": [0, 250, 500, 1000]
}
```

```
{
  "gobs": 9.80264, "phi": 40, "rho": 2.67,
  "points": [
    {"h": 0,    "normal_gravity": {...}, "free_air_correction": {"mgal": 0, ...},
     "bouguer_correction": {"mgal": 0, ...}, "bouguer_anomaly": {...}},
    ...
  ]
}
```

### 4.4 内置示例

```
GET /api/v1/example
```

### 4.5 错误响应

任何不合物理或格式不合法的输入都返回 `400`，并指明字段与原因，
绝不返回看似正常的结果：

```
400 {"error":{"field":"phi","reason":"纬度超出合理范围 [-90, 90] 度"}}
```

被拒绝的情形包括：

- 纬度超出 [-90, 90]；
- 密度 ≤ 0 或为 NaN/Inf；
- 缺失任一字段（注意 `h: 0` 合法，但缺 `h` 非法）；
- 高程超出明显非物理范围 [-12000, 10000] m；
- gobs ≤ 0 或为 NaN/Inf；
- 高程扫描序列为空、缺失或含非法高程（错误定位到 `heights[i]`）；
- 请求体不是合法 JSON、含未知字段或尾部有多余数据。

---

## 5. 本地构建、测试与运行

依赖已随仓库 `vendor/` 提交，构建与测试无需联网。

```bash
# 运行全部测试（可独立执行）
go test ./...
go test -v ./...

# 直接运行
go run ./cmd/server

# 构建静态二进制
CGO_ENABLED=0 go build -trimpath -o bin/gravity-server ./cmd/server
```

快速手验：

```bash
curl -s localhost:8080/api/v1/example
curl -s -X POST localhost:8080/api/v1/reduce \
  -H 'Content-Type: application/json' \
  -d '{"gobs":9.80264,"h":500,"phi":40,"rho":2.67}'
```

## 6. 容器构建

基于本地 Go 基础镜像 `golang:1.22-bookworm` 多阶段构建，
产物为跑在 `scratch` 上的静态二进制：

```bash
docker build -t gravity-reduction:latest .
docker run --rm -p 8080:8080 gravity-reduction:latest
```

容器内服务以非 root 用户（uid 65534）运行，监听 8080。

---

## 7. 代码结构（按职责拆分）

```
cmd/server/main.go              进程入口：HTTP server 生命周期
internal/gravity/units.go       mGal ↔ m/s² 单位换算（唯一入口）
internal/gravity/normal.go      国际正常重力公式 γ(φ)
internal/gravity/correction.go  自由空气改正、布格板改正
internal/gravity/reduction.go   单点归算合成 + 高程扫描逐点计算
internal/gravity/example.go     内置手工验算示例
internal/validate/validate.go   全部输入校验（带字段与原因）
internal/api/handler.go         HTTP 路由与 JSON 编解码
internal/api/example.go         示例接口
*_test.go                       物理关系、校验、HTTP 三层测试
```

## 8. 测试锁定的关键关系

- **高程归零**：h=0 时自由空气与布格板改正同时为 0，异常恰为 gobs−γ；
- **高程翻倍**：其余参数不变，两项高程改正都精确翻倍，γ 不变；
- **密度翻倍**：只有布格板改正翻倍，自由空气改正与 γ 毫不受影响；
- **只改纬度**：只有 γ 变化，两项高程改正保持不变；
- **自由空气符号方向**：正高程 FA 必须为正、负高程必须为负，
  且合成式严格为 gobs−γ+FA−B（符号取反即测试失败）；
- **扫描逐点真实计算**：每个扫描点与独立单点归算结果一致，
  γ 不随高程变，点列不退化为常数或写死直线。
