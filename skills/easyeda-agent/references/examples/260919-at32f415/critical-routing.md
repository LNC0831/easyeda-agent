# 样例：关键网络离线规划如何反向修正布局

## 来源与状态

- `考试说明.pdf` 第 5、8 页：晶振与 CAN 顶层、无过孔，晶振短直，CAN 保留 120Ω 跨接与近端 ESD。
- `原理图.pdf` 第 1 页：`OSC_IN/OSC_OUT`、`U5 → D1 → CN1` 与 R12 跨 CANH/CANL 的拓扑。
- 输入为第一轮参数化布局保存并整页刷新后的真实 pads/bbox，以及 6mil 间距、8mil 信号线宽。

状态：`offline-verified`。独立 subagent 生成并检查了两个可执行的离线路线，但没有写入 Web EDA。
结果证明“几何上能布通”仍可能是差布局；本样例的完成动作是把绕行原因反馈给布局参数，而不是
为了让工程看起来更完整而落下 77 段不理想走线。

## 开始状态与参数

目标 PCB 以 UUID 定位，不依赖 Board 显示名。开始时无走线，使用本轮读取的 pad 中心和矩形，
把其他网络 pad、器件和已规划轨迹按 6mil 净距膨胀为障碍。

| 参数 | 晶振 | CAN |
|---|---|---|
| 网络 | OSC_IN、OSC_OUT | CANH、CANL |
| 层/线宽/过孔 | TOP / 8mil / 0 | TOP / 8mil / 0 |
| 必达拓扑 | U6 ↔ X1 ↔ C20/C21 | U5 ↔ D1 ↔ CN1；R12 跨 H/L |
| 转角 | 0/45/90/135° | 0/45/90/135° |
| 计划哈希 | `2bdf39440d2d1a47dfb2e0de87dc5a5ec09f10e2f82a449a887478e0f6e6d92f` | 同一批 |

完整离线计划为本次开发记录，不随 Skill 打包；样例只保留能迁移的参数、结论和修法，避免把
58KB 的一次性绝对坐标塞进入口。迁移时必须从当前板重新读取 pads/bbox 并重算。

## 计划、观察与决定

离线规划先做 pad-normal escape，再在 45°格点上避开膨胀障碍，最后检查真实 pad 端点、板内、
异网轨迹间距、层和过孔数。检查覆盖 14 个必达端点，共 77 段、0 过孔、0 离线违规。

| 组 | 离线结果 | 观察 | 下一步 |
|---|---|---|---|
| 晶振 | 28 段；OSC_IN 882.7mil、OSC_OUT 269.4mil | U6.2/.3 与 X1.1/.3 左右次序反转，OSC_IN 被迫绕晶振区一大圈 | 旋转/重排 X1、C20、C21，保持两网引脚次序后重新求短解 |
| CAN | 49 段；保持 ESD 与 120Ω 拓扑 | D1 虽靠 CN1 外形，但未对齐 CANH/CANL pad，CANL 绕行 1422.4mil | 将 D1 横向对齐端子信号脚，保留 R12 跨接，再重算两支路 |

这一步没有把 `passed=true` 解释为“应该执行”。离线 passed 只证明给定障碍和检查器下没有已知
几何违规；短直、回流、EMC 和人工布局质量仍需看长度、拓扑与局部关系。

## 执行形态与回读

重排后重新生成段。每段映射到现有 typed CLI；下面只展示数据形态，不复制本轮绝对坐标：

```bash
easyeda pcb track --x1 <PAD_ESCAPE_X> --y1 <PAD_ESCAPE_Y> \
  --x2 <NEXT_X> --y2 <NEXT_Y> --layer 1 --width 8 --net OSC_IN \
  --doc <PCB_DOC_UUID> --project ceshi
```

按网络小批执行，每批后读取该网 tracks/vias；超时或部分写入时先回读，只 rip-up 当前失败网并
从新状态重算，不重放整组。晶振先于 CAN，二者通过后保存并做持久化回读：

```bash
easyeda pcb track-list --net OSC_IN --doc <PCB_DOC_UUID> --project ceshi
easyeda pcb track-list --net OSC_OUT --doc <PCB_DOC_UUID> --project ceshi
easyeda pcb via-list --doc <PCB_DOC_UUID> --project ceshi
easyeda pcb check --strict --doc <PCB_DOC_UUID> --project ceshi --json
easyeda pcb drc --doc <PCB_DOC_UUID> --project ceshi --json
easyeda pcb save --doc <PCB_DOC_UUID> --project ceshi
```

若 `doc reload` 在 Web 版持续显示加载动画，停止重试和现场写入，保存故障与当前对象证据，
将持久化验证标为 `incomplete`。先修复 typed reload/open，再重复 tracks/vias/check/DRC 读取；
禁止通过浏览器或工程树手工恢复。

## 暴露的工具缺口

- 当前 `route-short` 的 MST 只理解“同网端点”，不理解 `U5 → D1 → CN1`、R12 跨接等必经节点。
  后续接口需要显式 topology anchors 与 forbidden shortcuts，不能按最近距离偷连。
- 关键网需要每网 `topOnly`、`noVia`、pad-normal escape 等约束；这些应是计划输入与回读断言，
  不能靠 Agent 看完题目后记住。
- `pcb.line.create` 没有多段事务，超时可能已经写入前几段。工具必须返回部分执行证据；样例按
  net-scope 回读与删除，避免整板重放。
- DRC 规则配置、网络类成员及成员到 Track 规则的关联是三类事实，要分别回读。

## 验证边界

- 已验证：输入 pads/bbox、6mil 障碍、0/45/90/135°、14 个端点、TOP/8mil/0via 与两类拓扑的
  离线检查；独立规划明确给出不应执行的长度反例。
- 未验证：现场 track 创建、保存后持久化、官方 DRC、顶层地回流和重排后的最终短路线。
- 修法的验收不是“段数变少”本身：还要重新核对端点、拓扑、长度、净距、via 数和保存后对象。
