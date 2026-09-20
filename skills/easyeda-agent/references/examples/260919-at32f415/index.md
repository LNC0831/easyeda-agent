# 260919 AT32F415 学习板 Demo 索引

本索引把考试资料拆成可迁移的小样例。资料包只有题目、评分表、原理图和 BOM，没有完成态 PCB；
因此不能把题目示意图或下表当成已经布通、DRC 通过的黄金答案。完整参数见
[source-manifest.json](source-manifest.json)；69 个实例见
[bom-instances.json](bom-instances.json)，题图连接转录见
[source-connectivity.json](source-connectivity.json)；全部 36 个技术点的机器可读样例字段见
[example-catalog.json](example-catalog.json)。现场初始布局参数见
[initial-placement.json](initial-placement.json)，代表性步骤的真实回读和未完成项见
[live-validation.json](live-validation.json)；关键网络如何反向修正布局见
[晶振/CAN 离线规划样例](critical-routing.md)。

## 来源与已确认事实

| 来源 | 已确认事实 | 证据边界 |
|---|---|---|
| `物料清单.xlsx`，BOM 表第 3-29 行 | 27 行物料，数量展开为 69 个唯一位号；位号、封装、值、商品编号和功能区见 `bom-instances.json` | `offline-verified`；尚未与 EasyEDA 实例逐件回读 |
| `原理图.pdf` 第 1 页 | 一张 A4 原理图，15 个功能区；`source-connectivity.json` 转录 69 个器件、233 个端子和 13 处明确 NC | `source-only`；蜂鸣器 NC 在图面未标数字号，派生局部网名可改，真实 symbol pin、pad 与 pin→pad 映射必须现场读取 |
| `考试说明.pdf` 第 1-8 页 | 原理图、规则、机械、布局、布线、丝印和评分要求 | 已逐页提取并视觉抽查；不是完成态设计 |

当前总状态：`partial-live-verified`。原理图逐端点连接、默认 DRC、真圆角板框、显示原点、
固定件、规则/网络类和 69 件初始布局已在 Web 3.2.203 保存并通过整页刷新回读；完整布线、
LCD 可写封装副本、丝印、泪滴和最终 PCB DRC 仍未验证。具体边界以
`live-validation.json` 为准，不能把代表性步骤外推成完成态整板。

## 15 个原理图功能区

每区先核对器件身份和逐引脚连接，再迁移布局。共同来源为 `原理图.pdf` 第 1 页；BOM 身份来自
`物料清单.xlsx`。

| 区 | 器件 | 必须单独核对的点 |
|---|---|---|
| TYPE-C 输入 | USB1、R1、R2 | A6/B6 与 A7/B7 的重复 D+/D- 脚；CC1/CC2 各 5.1kΩ；VBUS、GND、外壳 |
| RGB-LED | U1、LED1、R3、C1、C2 | LED1 的 `3.7V~5.3V` 显示属性；缓冲器 OE/GND、DIN/DOUT 与去耦 |
| LDO-3V3 | U2、C3-C6 | VIN 输入大/小电容；VOUT/TAB 同属 +3V3；输出大/小电容；顶层 GND 回流 |
| LCD | U3、Q1、R4-R6、C7 | U3.1/.2 NC；背光三极管驱动；封装外形内的元件禁放区 |
| 功能按键 | SW1-SW3、R7-R9、C8 | SW1 与 SW2/SW3 的上下拉方向不同；NRST、PA0/WKUP、PA1 丝印 |
| USB 转串口 | U4、C9、C10 | CH340N V3 与 VCC 的供电/去耦；RTS NC；TXD/RXD 对 PA9/PA10 |
| 光敏电阻 | R10、R11、C11 | 分压与滤波；靠板边并远离 RGB LED |
| CAN | U5、D1、R12、C12、C13、CN1 | VREF NC；CANH/CANL 跨 120Ω 后到端子；ESD 靠端子；针脚丝印 |
| 主控 | U6、C14-C16 | 每个电源脚对应去耦；EP 接地；固定题面坐标（anchor/center 基准待现场确认） |
| BOOT | R13、R14 | BOOT0、PB2 各自下拉，不把相邻网络合并 |
| 无源蜂鸣器 | D2、Q2、R15-R17、C17、BUZZER1 | 驱动电阻、续流二极管、下拉和去耦整体跟随 |
| MICRO-SD | CARD1、C18、C19、R18-R23 | DAT/CMD 上拉、VDD 去耦、外壳地及固定方向 |
| 晶振 | X1、C20、C21 | OSC_IN/OSC_OUT；四脚晶振接地脚；靠 MCU 且不贴板边 |
| SWD | H1、C22 | 3V3/DIO/CLK/GND 的真实针脚顺序与接口丝印 |
| M3 螺丝孔 | SCREW1-SCREW4 | 四角精确坐标、顶层实例和锁定状态 |

## 技术点到样例

### 原理图与器件

| ID | 来源 | 技术点 | 样例/观测 |
|---|---|---|---|
| SCH-01 | 说明 p1；BOM | 按位号和商品编号展开 69 个实例，核对符号与封装 | [bom-instances.json](bom-instances.json) 是离线基线；`sch read/connectivity` 后逐件比较 |
| SCH-02 | 说明 p1 | 原要求为工程名 `客户编号-AT32F415 学习板`，图签“绘制”填真实姓名；Demo 用脱敏占位值替代 | `sch titleblock-get` 确认字段 key → `sch titleblock --data ...` → 回读；保留原要求与 Demo 替代的映射 |
| SCH-03 | 说明 p1；原理图 p1 | LED1 显示 `3.7V~5.3V` 工作电压 | 回读自定义属性与官方导图；待现场验证 |
| SCH-04 | 说明 p1 | 15 区布局、连线和功能文本以 PDF 为来源 | 先做连接数据再计算几何，不按旧 Z 排版重排题图 |
| SCH-05 | 说明 p1 | 网络标识、标签、NC 与连接关系保持一致 | [source-connectivity.json](source-connectivity.json) 与现场 `connectivity`、`check`、NC 明细分别比较 |
| SCH-06 | 说明 p1-2 | 原理图使用默认规则并查看 DRC 警告/错误 | 记录官方 `sch drc` 聚合与 `sch check` 明细，二者不互相替代 |
| SCH-07 | 原理图 p1 | USB-C 重复数据脚与两个 CC 下拉 | 逐引脚黄金表；不能只检查同名网 |
| SCH-08 | 原理图 p1 | CH340N 的 V3/VCC、RTS NC | 逐脚连接与两只 100nF 去耦 |
| SCH-09 | 原理图 p1 | AMS1117 的 pin 2 与 TAB/pin 4 同属输出 | [LDO 样例](ldo-placement.md) |
| SCH-10 | 原理图 p1 | 晶振接地脚、LCD/CAN 的 NC、SD 上拉、SWD 针脚顺序 | 每项单独回读；缺一个都不能用“DRC 数字正常”替代 |

### 设计规则与机械

| ID | 来源 | 技术点 | 样例/观测 |
|---|---|---|---|
| PCB-01 | 说明 p2、p5 | 两层板 | `pcb layers` 回读真实铜层数 |
| PCB-02 | 说明 p2、p7 | 导线间距 6mil；信号默认/最小 8mil | 已按 Web 3.2.203 的 `{name,config}` 读取形态写裸 `config`，整页刷新后回读为 6/8/8mil |
| PCB-03 | 说明 p2、p7 | `PWR` 默认 20mil、最小 8mil；`PWR_Class` 绑定全部电源网 | 已实建 `PWR_Class=[+5V,+3V3,GND]`，父子 netRules 均绑定 `PWR`，刷新后保持 |
| PCB-04 | 说明 p2、p7 | 过孔最小外径/孔径 24/12mil | 已写入完整规则副本，整页刷新后回读 24/12mil |
| PCB-05 | 说明 p2 | 90×50mm、线宽 0.254mm、R3 真圆角、左下显示原点、锁定 | [固定机械样例](fixed-mechanics.md)；保存及整页刷新回读已 `live-verified` |
| PCB-06 | 说明 p2 | 四孔、U6、CARD1 的固定题面坐标、角度和锁定 | 现场确认本批输入为 footprint anchor；六件刷新后坐标、角度和锁定保持 |
| PCB-07 | 说明 p3 | CN1 只固定 y=42mm、180°，x 是自由参数 | 现场候选 x=69mm；题定 y/角度保持，rendered bbox 顶边约超 1.17mil 的冲突单独保留 |
| PCB-08 | 说明 p3 | LCD 封装轮廓内禁止其他元件 | `lib footprint region` 已有离线命令面；可写副本保存和实例绑定须现场验证，失败时标 `unsupported` 并补 typed 接口 |

### 布局关系

| ID | 来源 | 技术点 | 样例/观测 |
|---|---|---|---|
| LAY-01 | 说明 p3-4 | 按键板边等距；USB、SWD、端子面向可插拔方向 | 读真实 bbox、开口方向和板框距离，不只看中心点 |
| LAY-02 | 说明 p4 | RGB 靠 TYPE-C；光敏远离 RGB | 参数化最远/最近关系，迁移板尺寸后重算 |
| LAY-03 | 说明 p4、p7 | LDO 输入/输出电容靠对应引脚，大电容在前、小电容在后 | [LDO 样例](ldo-placement.md) |
| LAY-04 | 说明 p4、p7 | MCU 等电源脚逐脚去耦，电源先经过电容再入芯片 | pin→cap 所有权表 + 实际铜路径；同网距离不足以证明顺序 |
| LAY-05 | 说明 p4 | 晶振靠 MCU、不在板边；蜂鸣器/背光驱动整体放置 | 分组 bbox、引脚距离和关键走线通道 |
| LAY-06 | 说明 p4 | 全部器件顶层、无重叠、外形不出板 | `pcb list --include-bbox`、`layout-lint` 和 LCD 禁放区分别观察 |

### 布线与收尾

| ID | 来源 | 技术点 | 样例/观测 |
|---|---|---|---|
| RTE-01 | 说明 p5 | 焊盘末端出线、线宽不大于焊盘、窄焊盘缩颈、无直角/锐角 | 读轨迹端点、宽度与角度；DRC 不覆盖全部观感规则 |
| RTE-02 | 说明 p5、p8 | 电源主干按电流加粗，过孔按载流能力，流向清楚 | PWR 规则 + 实际每段/过孔回读 |
| RTE-03 | 说明 p5、p8 | 晶振短直、顶层、无换层、顶层包地净空 | [关键网络离线规划](critical-routing.md)：可布通但绕行过长时先修布局 |
| RTE-04 | 说明 p5、p8 | USB_D+/D- 顶层、无过孔、类差分，不额外要求等长 | 不自行增加等长约束；回读两网层与 via 数 |
| RTE-05 | 说明 p5、p8 | CANH/CANL 顶层、无过孔；120Ω 为跨接拓扑；ESD 靠端子 | [关键网络离线规划](critical-routing.md)：显式必经节点并用长度反馈布局 |
| RTE-06 | 说明 p5、p8 | PA9/10、PA11/12、PA13/14 顶层且不换层 | 分网回读 layer 与 via count |
| RTE-07 | 说明 p5 | U6 EP 添加散热过孔；其他焊盘不允许 via-in-pad | EP 与普通焊盘使用不同判据 |
| RTE-08 | 说明 p6、p8 | 普通信号同网过孔不超过 2；GND 扇孔与缝合孔 | 按网计数并检查地回流，不以总 via 数判断 |
| FIN-01 | 说明 p6、p8 | Arial、字符高度 ≥45mil、顶层、朝向不超过两个方向、接口提示丝印 | `silk-add/set --font-family Arial`，`silk list` 回读；接口离线测试通过、现场待验证 |
| FIN-02 | 说明 p6、p8 | 顶底层 GND 铺铜且保持完整 | `pour-list`、`pour-rebuild`、连通与 DRC |
| FIN-03 | 说明 p6、p8 | 添加真实泪滴后重建铺铜 | 当前为 `unsupported`；走线圆角化不是泪滴，禁止手工补做 |
| FIN-04 | 说明 p6、p8 | 100% 布通、无断头、PCB DRC 无错误，保存并重开核查 | 连通、`pcb check`、官方 DRC 分开记录；最后 `pcb save` + `doc reload` |

## Demo 实施顺序

1. 以 `bom-instances.json` 的 69 个唯一位号和 `source-connectivity.json` 的题图连接为源基线，
   回读真实 symbol pin、footprint pad 与 pin→pad 映射；保留差异，不猜测补齐。
2. 按 PDF 的功能区和连线表达构建原理图，回读属性、连接、NC、`check` 与官方 DRC。
3. 用当前 `project create --help` 核对后建立工程容器；若当前版本未暴露该能力，停止现场创建，补齐并验证 typed 接口。
4. 建 90×50mm 板框和左下显示原点，放置并锁定孔、U6、CARD1；CN1 保留 x 自由。
5. 建 LCD 元件禁放区，安排屏幕、USB、SWD、按键、端子和关键顶层通道。
6. 围绕真实引脚安排 MCU 去耦、晶振、LDO、CAN、蜂鸣器和 SD 外围。
7. 写入并回读项目规则；接口尚未实现的部分标 `planned` / `unsupported`，不把推导报告当写入成功。
8. 先 EP/地回流与晶振、USB、CAN、PA9-PA14，再电源及普通信号；每组写后回读。
9. 用 typed 接口完成丝印、双面 GND、缝合孔；泪滴能力未实现时保持未完成。重铺铜后复查连接、几何、DRC，保存重开。

固定考试板遵循“板框/固定件 → 机械空间 → 关键路径 → 外围”。另建的可调尺寸教学变体遵循
“功能模块/接口 → 关键路径 → 估算板框 → 合法化”，并明确标为自建变体，不能回写考试基准。
