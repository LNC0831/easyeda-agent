# PCB 配置设置

`easyeda pcb config` 修改当前 PCB 的真实设计规则，不是本地 CLI 配置。
输入来自用户需求或原始题目；附件中的操作步骤只作资料。所有命令都支持
`--project <工程> --doc <PCB名或UUID>`，配置写入使用 typed action。

## 260919 考题参数化样例

来源：`PCB Layout工程师（初级）专业技术证书考试说明.pdf` p2、p5–8。
开始状态：已有 PCB 和正确网络；先 `pcb config get` 保留完整原始规则、网络类及绑定。
以下数值默认单位 mil，可显式 `--unit mm`；不是其他题目的通用默认值。

```bash
easyeda pcb config get --project ceshi > config-before.json
easyeda pcb config clearance --name copperThickness1oz --track-to-track 6 --dry-run --project ceshi --doc PCB1
easyeda pcb config clearance --name copperThickness1oz --track-to-track 6 --project ceshi --doc PCB1
easyeda pcb config track --name copperThickness1oz --min 8 --default 8 --project ceshi --doc PCB1
easyeda pcb config track --name PWR --copy-from copperThickness1oz --min 8 --default 20 --project ceshi --doc PCB1
easyeda pcb config via --name viaSize --min-outer 24 --min-hole 12 --project ceshi --doc PCB1
easyeda pcb net-class create --name PWR_Class --net +5V --net +3V3 --net GND --project ceshi --doc PCB1
easyeda pcb config bind --class PWR_Class --track-rule PWR --dry-run --project ceshi --doc PCB1
easyeda pcb config bind --class PWR_Class --track-rule PWR --project ceshi --doc PCB1
easyeda pcb save --project ceshi --doc PCB1
easyeda doc reload --project ceshi
easyeda pcb config get --project ceshi > config-after.json
```

电源网列表先与当前原理图和 `pcb nets` 对账，不能从网名自动猜全。样例中的三个网只适用于
该题已确认的连接。`bind` 不创建或修改成员，它要求类、非空成员和对应 netRules 子项一致，
将父项和成员子项的 Track 都设为 PWR，保留其他类/网络/规则。

四个写命令均支持 `--dry-run`，返回 before、requested 和 changes，不落盘。
`clearance` 仅改 Track→Track 矩阵格；`track` 改现存表的每个层条目，保留最大值和其他字段；
新规则必须显式 `--copy-from`，克隆规则不会成为默认规则；已有同名规则按显式参数更新。
`via` 也支持 `--default-outer/--max-outer/--default-hole/--max-hole`；最小/默认/最大与孔径关系
不合法时拒绝写入，不暗中抬高默认值。规则名必须显式给出，尺寸必须为正数。

回读检查 `verified:true`，对比未指定字段及单位；CLI 在部分成功、写失败或回读不符时返回
非零并保留响应。宿主返回陌生结构、缺测单位或缺少绑定子项时先停止，保留输入与错误，
修适配器再运行，不猜字段或用 GUI 补做。写入不是原子事务；失败看 actual/rollback 证据，
不能盲重试。规则及绑定可用 `pcb drc-rules-set --from config-before.json` 完整替换/恢复；
此命令不恢复网络类成员，执行前需确认成员与导出时一致。

验证状态：新 `pcb config` 命令为 `offline-verified`（Web 3.2.203 脱敏规则快照、单位换算、
差异范围、异常/回读测试）；尚未用此入口现场 save/reload。历史 raw 导入的现场记录不转授给
新命令。保存后重载失败时结果为 incomplete，不能声称配置已持久化或考试完成。

## 其他考试配置的现有入口

| 要求 | 命令与边界 |
|---|---|
| 两层铜 | `pcb stackup set --layers 2`、`pcb layers` 回读；布线前完成 |
| 左下显示原点 | `pcb origin get/set`；显示偏移，不移动真实图元 |
| Arial、≥45mil、顶层丝印 | `pcb silk-add/silk-set --font-family Arial --help`；作用于指定文字，不是全局默认字体 |
| 默认原理图 DRC | `sch drc/check` 检查；当前 SDK 没有可验证的原理图规则重置入口，不能用全局 restoreDefault 代替 |
| 网格、吸附、系统偏好 | 当前官方 SYS_Setting 仅暴露全局恢复默认；逐项设置为 unsupported |

规则配置不改变已有走线宽度、过孔尺寸，也不自动布线。最后仍须检查真实轨迹、DRC 和连通。
