# 原理图通用算法验证 — v1.5.0-dev.9

2026-09-17。目标是通用算法，不是修好某一张原理图。离线 P1/P2 已通过，统一 dev.9
本地包已经构建但**没有安装**，当前 CLI、Skill、daemon 和
Web Connector 均未在本轮替换，也没有现场 Apply。本记录不是 v1.5.0 发布验收。
规范唯一来源为 [Skill 数据驱动架构](../skills/easyeda-agent/references/schematic-data.md)。

## 已实现的通用契约

- 核心/外围显式归属；专属外围经真实线树连接，不能以同名标签或同框替代。
- 实测引脚外向方向贯穿刚体变换、候选筛选、执行守卫；器件、NC、位号及真实参数保全。
- 区分无接点内部 X 与端点/T/重叠接触；同网 X 仍不合并物理线岛。
  写后另核对物理分区，几何覆盖相同不能掩盖意外合并或断开。
- 几何失败记录两侧阻挡所有者，包括原导线的所属器件。归属完整、没有未来可用宿主时
  跳过无关 checkpoint，回退真正相关的宿主或阻挡器件；未知归属保守探索。
- 直线/折线失败后使用 5 raw 方向 A*；支持真实线岛到线树中段、X/T 分离、整网撤线重布、
  direct 局部前沿和最多 40 raw 的 attachment 刚体迁移。已连接线树可从真实中段安全命名，
  但标签不能替代 direct 连接。
- `layout-plan --report` 在成功和失败时都输出独立机器报告：源哈希、算法版本、阶段、
  搜索总预算/回退、放置冲突、路由冲突、各允许姿态的失败诊断。保留多重错误链；
  失败非零退出，不输出半成品，也不覆盖上一份合法布局来冒充成功。
- 旧 group-move/disconnect 对尚未安全支持的复杂拓扑写前拒绝，不用错误段解析继续写入。
- `sch layout-edit` 将一个 zone 的器件、引脚、位号、内部导线和标签统一表示为核心相对坐标；
  核心移动只应用一次区级平移。刚体目标碰撞时固定用户给定核心位置，仅重算本区外围，
  其他功能区作为固定障碍并保持不变。
- 标签显式区分 `pin` 与 `wire_tree` 锚定。单脚修复只沿实测引脚外向轴寻找合法长度；
  daemon 写前校验完整目标、旧线/标签、scene 指纹和标签本体旋转，串行替换后逐对象回读。
  范围外旧错误可保留但签名必须完全一致；部分写入不重试、不声称回滚。

## 回归证据

测试均使用纯计算或模拟宿主；真实反例仅用于校准语义，不按工程名、型号、位号或坐标特判。

| 回归族 | 检查内容 |
|---|---|
| `sch_layout_contract_test.go` | 不透明 ID/位号/网名重命名、源平移、四旋转与镜像、NC、确定性、输入不可变、标签替代实线负例 |
| `sch_layout_placement_conflict_test.go` | 封闭宿主失败、真实宿主回退成功、无关 checkpoint 跳过、未来同网宿主救活、预算与四旋转/平移 |
| `sch_layout_obstruction_owners_test.go` | 原导线所有者不会漏掉；移动该所有者可解除阻挡；未知归属不误称完整 |
| `sch_layout_maze_test.go` / `sch_layout_frontier_test.go` | 多折绕障、扩大边界、不同出脚方向、线树接入、X 穿越不造接点、局部前沿与节点预算终止 |
| `sch_layout_naming_retry_test.go` / `sch_layout_repair_test.go` | 拥挤端点改从真实线树中段命名、异网端点/T 拒绝、40 raw 有界刚体回退与输入不可变 |
| `sch_layout_report_test.go` / `sch_layout_feasibility_test.go` | 真实冲突与总预算报告、多重包装、早期失败姿态、报告 IO 失败、输入/输出别名保护 |
| `sch_wire_contact_test.go` / `internal/schguard` | ABAB 边界引脚、X/T、共线 waypoint、接触分区、方向、NC 和缺证据拒绝 |
| `schematic-wire-topology.test.ts` | 官方独立段解码、交叉证据、实际接触、级联保全及旧操作安全拒绝 |
| `sch_layout_edit_test.go` / `sch_layout_marker_anchor_test.go` | 核心与多级外围单次跟随、碰撞后固定核心局部重排、区外不变、跨区线树拒绝、重命名/输入不可变、预算终止、D1.3 外向支路及 pin/wire-tree 锚定 |
| `schematic_pin_repair_test.go` | 旧错误存在时的作用域替换、陈旧指纹、篡改标签朝向、部分删除、连接超时/partial、范围外对象变化全部失败关闭 |

宿主回退合成例在原 64000 候选上限内以 10282 次候选、2 次回退得到完整方案；
这是该合成用例的观测，不是所有电路的性能保证。

已通过：

- `go test ./...` 与 `make lint-test`；通用契约、回退、报告、方向真值表和规则 fixture 全绿。
- 保留的原始 P1 三 zone（源 SHA-256 `2128532c…`）全部完成：POWER_ENTRY/USB_SERIAL/
  BUCK_3V3 分别使用 6927/4469/20597 个候选；A* 展开 1682/0/38 个节点，无重布。
- 保留的原始 P2 五 zone（源 SHA-256 `c8b88e44…`）全部完成；仅 MCU 使用 A* 53 个节点，
  无重布。dev.7 的 184652 节点/4 次重布是另一个保留压力输入，不与本次原始输入混称。
- 连接器完整测试 328 项，0 失败、0 跳过；`npm run typecheck`。
- `make lint-test blocks-audit modules-audit`；1027 个块引脚引用无缺失/未知，35 条模块记录通过。
- `make skill-check release-script-test`；63 项脚本测试通过。
- `git diff --check`。
- `make local-build VERSION=v1.5.0-dev.9 DIST=dist/local-v1.5.0-dev.9`；全部资产
  checksum 通过。打包后的 Darwin arm64 CLI 重放 P1/P2，输出 SHA-256 分别为
  `0ee8864423c5c68d310db13e4fbef9b6b97c4cad27f54f1454121c27dcc41c24` 与
  `8e53d6e4e240e70a16bec8c539212a5c588d262856e20d68b0a5535e4db8ee9b`，
  和当前源码结果逐字节一致；输入哈希与报告中的 `sourceSha256` 一致。
- dev.9 尚未替换当前运行时。后续安装 CLI/Skill/daemon/Connector 后，必须结束安装会话，
  由下一全新会话执行首条本地版本门禁，才能生成新鲜现场快照并 Apply。

这些 Go/连接器测试由已有 CI 的 `make test` 和 `npm test` 自动发现，不依赖 Agent
记住单独跑某个测试文件。以上是本地证据；在对应提交的远端 CI 实际完成前不声称 CI 已通过。

## 明确边界

有界窗口、姿态菜单与预算不是完备搜索；仍可能重复探索局部预算窗口。失败不证明全局无解，
不能靠扩大预算、放松硬约束或手改最终坐标宣称解决。外围语义归属仍需显式输入。
宿主接触语义现场证据限于已测 EasyEDA Pro 3.2.186；未知宿主行为须完整回读，不推定兼容。
离线实例保全/回读守卫测试不等于新运行时现场验收；PCB 与正式发布均不在本轮执行范围。
