<p align="center">
  <img src="docs/assets/easyeda-agent-logo.png" width="96" alt="easyeda-agent logo" />
</p>

<h1 align="center">easyeda-agent</h1>

<p align="center">
  面向 EasyEDA(嘉立创EDA专业版)的 AI 原生自动化层
</p>

<p align="center">
  <a href="https://github.com/zhoushoujianwork/easyeda-agent"><b>GitHub</b></a> ·
  <a href="https://jlc-ext.com/item/zhoushoujian/easyeda-agent-connector"><b>立创插件市场</b></a> ·
  <a href="README.en.md">English</a>
</p>

![easyeda-agent workflow](docs/assets/easyeda-agent-workflow.svg)

> **1.4.5 版本。** 原理图以器件、引脚与连接图为依据,先在本地数据中设计
> Lib 电路与几何,再通过 `sch compose` 组合单页、`sch apply` 顺序执行并回读验证。
> 发布状态、构建步骤与尚未完成的验收见 [1.4 发布与验证](docs/release-1.4.md)。

`easyeda-agent` 把官方 EasyEDA 扩展 API 变成一套**有类型、可观测、Skill 友好**的系统。EasyEDA 插件保持极薄——它连到本地 agent、只执行被批准的动作;Go CLI/daemon 掌管协议、状态、产物、校验和面向用户的工作流。

## 为什么做这个

上游 `run-api-gateway` 证明了关键入口:代码能跑在 EasyEDA 内、访问官方 `eda` 对象。但它把「裸 JavaScript 执行」当作主工作流——强大,但对 AI agent 太脆弱。

本项目的连接器是真实可用的:daemon **默认固定监听单端口 `60832`**(不外溢、被占用时自动接管旧 easyeda daemon)、连接器锁定该端口、校验握手、**自愈重连**、把一套**有类型的动作目录**分发到官方 `eda.*` API。`debug.exec_js` 保留为任务范围内的临时调试入口。

- **Skill** 描述专家工作流和护栏;
- **Go CLI/daemon** 暴露稳定的 typed actions;
- **EasyEDA 连接器插件** 只做到官方 `eda.*` 的桥接;
- 产物、截图、DRC 结果、审计日志都是一等输出。

## 工作原理

- Skill 或人跑一条 `easyeda` 命令;
- Go CLI 校验输入、把 typed action 提交给本地 daemon;
- daemon 跟踪已连接的 EasyEDA 窗口、经 WebSocket 路由每个动作、记录审计日志/产物/校验结果;
- 连接器扩展跑在 EasyEDA 内、调用官方 `eda.*` API;
- 结构化结果回流到 CLI 和 Skill,下一步基于**真实编辑器状态**来规划。

动作目录已覆盖原理图、PCB、文档导航、板级绑定、产物导出、诊断。完整清单与路线图见 [docs/FEATURES.md](docs/FEATURES.md)。

## 站在巨人的肩膀上

我们不重造轮子,而是把**成熟的一层层能力叠起来**,让 AI agent 直接可用:

- **官方 `eda.*` API** —— 嘉立创 EDA 专业版自己暴露的 86 个命名空间,是真正的能力底座;
- **上游 `run-api-gateway`** —— 证明了「代码能跑在 EasyEDA 内、访问 `eda` 对象」这条关键入口;
- **成熟的 AI Agent Skill 范式** —— 用 Skill 描述专家工作流 + 护栏,用 typed action 让每一步**可观测、可验收、可回放**,而不是把「裸 JS 执行」丢给模型硬扛。

在这三层之上,easyeda-agent 补齐了工程化的中间层:自愈连接器、有类型的动作目录、真实 bbox 校验、门控设计流程,以及下面这个**核心特色**——电路块库。

## 核心能力 & 特色

**能力总览**(完整清单见 [docs/FEATURES.md](docs/FEATURES.md)):

| 能力域 | 做什么 |
|---|---|
| **电路块库(旗舰特色)** | 社区共建、署名可追的**成熟外设电路库**(`easyeda blocks`,**37 块:19 ready / 13 verified / 5 draft**):CH340 USB 串口、ESP32 自动下载、按键去抖、USB-HUB、降压…`sch block-apply` **一条命令放件 + 连线 + 网表对账**,照抄拓扑、只重绑引脚网络即可复用 |
| 原理图 | 原始快照 → canonical/核心外围归属/约束 → 区内与纸张两层计算 → 固定转换/Apply → 原始数据回读；失败回源重算。[架构基准](skills/easyeda-agent/references/schematic-data.md#数据驱动架构基准) |
| 机械门禁与审计 | 本地数据检查、Apply 后逐脚/网络/NC/几何回读;`sch gate --strict` 四阶段(layout-lint→check→bridge-check→drc);跨页网名审计 `sch nets --strict` + 块对账 `sch reconcile` |
| PCB | 自动布局、板框、禁布区、规则感知短线布线、4 层电源平面、铺铜、丝印避让、DRC/`pcb check` |
| 设计流程 | 从**客户口吻需求**到成品的门控主脊(S0–S6 + P0–P10),里程碑确认,存盘检查点 |
| 产物 | BOM(补 LCSC C 号)、网表、导出、原生截图、审计日志、录制→回放 |

### 特色:电路块库(一次贡献,永久收益)

**固定模块的外设电路可以直接照抄。** ESP32 自动下载电路、CH340 USB 烧录、按键去抖、
USB-HUB…这些电路的**内部拓扑是死的**,每次重画等于重趟坑。电路块库把它们
沉淀成**验证过的、可复用的电路块**(当前 **37 块:19 ready / 13 verified / 5 draft**)——
`sch block-apply` 一条命令完成**放件 + 连线 + 网表对账**,你只需重绑对外的几根线(ports)
到主控网络,引脚用**功能名**引用所以**零改号**,器件直接指回标准器件库(BOM 就绪)。

- **社区共建 + 署名可追**:每个块带 `author`/`contributors`,**一次学习贡献、永久收益**;
- **验证门禁**:块必须跑过 `place → wire → check → DRC=0` 才入库,不是「看着对」的散文堆;
- **三维知识**:器件(可替换选择)+ 原理图链接注意 + PCB 布局电气特性,一块讲全;
- **AI 直接消费**:agent 放外设前先查块库,命中即抄,省掉一整个模块的选型与接线。

> 库已**内嵌进 CLI**:`easyeda blocks ls/show/search` 离线可查(无需 daemon/窗口) ·
> 贡献指南
> [`standard-blocks-contributing.md`](skills/easyeda-agent/references/standard-blocks-contributing.md)

## 安装

> **完整上手 & 使用注意事项见 [快速开始 →](docs/quick-start.md)** —— 三要素
> (CLI / 连接器 `.eext` / Skill)的安装、版本对齐、启动 daemon、升级纪律
> 与常见卡点速查,一页讲清。下面是精简版。

easyeda-agent 有三个必须配套的组成部分:CLI/daemon、连接器 `.eext` 插件和
`easyeda-agent` Skill；EasyEDA Pro 是运行它们的宿主,需开启「允许外部交互」。升级时
CLI、Skill 与运行中的 daemon 必须精确等于 latest；Connector 只在跨 major/minor 兼容线时
升级，同一兼容线里的市场版 patch 滞后不影响使用。

先装 `easyeda` CLI/daemon,再装 EasyEDA 连接器 —— 两条通道任选:安装器会打印 GitHub Release `.eext` 下载地址,或从[**立创官方插件市场**](https://jlc-ext.com/item/zhoushoujian/easyeda-agent-connector)一键安装。CLI/daemon 与 Skill 精确同版；Connector 只需 major.minor 相同，市场版仅落后 patch 时无需升级:

> **ℹ️ 插件更名说明(2026-08)**:应市场管理规范要求,插件**显示名**改为
> **EDA Agent Connector**(不再含 "easyeda" 字样)。经与市场管理员确认,内部包名
> `easyeda-agent-connector` 与 uuid 均**保持不变**,同一条目重新上传即可 ——
> 已装用户的原地自动更新不受影响,无需任何操作。

```bash
curl -fsSL https://raw.githubusercontent.com/zhoushoujianwork/easyeda-agent/main/install.sh | bash
```

一键脚本会：安装/更新 `easyeda` CLI/daemon;自动检测已安装的客户端并把 `easyeda-agent` skill 安装/更新到对应目录 —— Codex(`~/.codex/skills/easyeda-agent`)、Codex Desktop 共享目录(`~/.agents/skills/easyeda-agent`)、Claude Code(`~/.claude/skills/easyeda-agent`);打印连接器 `.eext` 导入地址。

**装过之后升级不必再跑脚本 —— 用 `easyeda update`:**

```bash
easyeda update              # CLI 二进制(sha256 校验 + 原子替换)+ skill 目录 → latest
easyeda update --check      # 只读:cli / skill / connector 三方版本对齐表
easyeda update --check --exit-code   # CLI/Skill/daemon 非 latest 或 Connector 跨兼容线则退出 10
easyeda update --version <x.y.z>     # 钉版本;--skill-only / --cli-only 缩范围
```

GitHub 大资产连续三次下载失败时，安装器和 `easyeda update` 会尝试
`https://gh-proxy.com/` 传输，但只在已从 GitHub 取得 `checksums.txt` 时启用，并仍按该
SHA-256 校验。可用 `EASYEDA_GITHUB_PROXY=https://your-mirror/{url}` 换镜像，或设为 `off`
禁用；镜像只负责可用性，不是供应链信任根。

连接器 `.eext` 不在自动升级范围内(侧载无原地更新)—— 同 major.minor 的 patch 差异直接兼容；
只有跨 minor/major 时 `update` 才会阻断并打印重导地址。
dev 构建(git-describe 版本号)默认不覆盖,`--force` 才强升;二进制在 root 目录时用 `sudo easyeda update`。

可用环境变量控制 skill 安装:

```bash
curl -fsSL .../install.sh | EASYEDA_INSTALL_SKILLS=codex,agents,claude bash  # 指定目标
curl -fsSL .../install.sh | EASYEDA_INSTALL_SKILLS=none bash  # 跳过 skill
curl -fsSL .../install.sh | EASYEDA_SKILL_PRESERVE=1 bash  # 保留本地改动
curl -fsSL .../install.sh | EASYEDA_VERSION='<vX.Y.Z>' bash  # 指定版本(跳过 API 查询)
```

**遇到 `403` / GitHub API 限流**:脚本默认要调一次 `api.github.com` 解析 latest
release,匿名调用每个 IP 每小时只有 60 次 —— 公司出口 / NAT / CI 很容易撞满。两条
出路(脚本报错时也会打印):

```bash
export GITHUB_TOKEN=<token>   # 或 GH_TOKEN;已登录 gh CLI 时会自动取 `gh auth token`
gh auth login                 # 等价做法,额度提升到 5000/小时

curl -fsSL .../install.sh | EASYEDA_VERSION='<vX.Y.Z>' bash   # 或者直接锁版本,完全不碰 API
```

可用 tag 见 [Releases](https://github.com/zhoushoujianwork/easyeda-agent/releases)。

Skill slug 为 `easyeda-agent`(后缀有意为之,区分于官方 EasyEDA 工具)。只从 registry 装 skill:

```bash
# ClawHub(make release 时自动同步发布,版本与 repo 对齐)
clawhub install easyeda-agent
```

> SkillHub 另有[官方 CLI](https://skillhub.cn),与其他同名 `skillhub` 工具不兼容。
> 需要与 CLI/连接器保持同版时,使用上面的一键安装器或 GitHub Release 的 `skills.tar.gz`。

> EasyEDA 需开启「**允许外部交互**」,连接器的 WebSocket 才能连到本地 daemon。

### 给 AI Agent 的推荐引导 Prompt

把下面内容连同具体设计需求交给 Agent。它要求 Agent 先确认工具链和数据证据，再写入
EDA，避免直接从截图猜接或在页面上反复试摆：

```text
请使用 easyeda-agent 完成 EasyEDA Pro 任务。

开始前先确认三个组成部分处于兼容的发布版本：
1. easyeda CLI/daemon
2. easyeda-agent Skill
3. EDA Agent Connector 插件

把 easyeda update --check --exit-code 作为当前会话第一条命令。只有 CLI、Skill、运行中
daemon 可验证且精确等于 GitHub latest，且所有 Connector 与 latest 同 major.minor 时继续。
否则运行 easyeda update 并重启 daemon；纯 patch 更新保留现有 Connector，不重开 EasyEDA。
仅当 Connector 跨 minor/major 时安装命令所示 GitHub Release 的
easyeda-agent-connector.eext，保存文档并完全退出、重开 EasyEDA。任何组件升级后立即结束
当前 Agent 会话并新开会话，从版本检查重新开始。确认已开启“允许外部交互”，再运行
easyeda health 核对目标工程、页面和版本。

先读 Skill 的 schematic-data.md「数据驱动架构基准」。保留官方原始快照，在源数据副本
明确 canonical 连接、核心/外围归属、参考引脚与约束；用 layout-plan --zones、
layout-sheet-plan、layout-render 计算并验证，已确认页用 compose --layout-page 固定转换。
问题由数据检查发现，修源数据/采集/算法后重算，不以现场逐件试摆或手改队列兜底。
Apply 后回读器件、pin→net/NC、真实直连、位号和框/标题，逐页严格门禁并显式保存。
位号参与遮挡/入框，型号/参数等非位号属性文字排除布局检查；截图只辅助发现规则遗漏。
不把同网/同框、高分、缺测或未完成溯源当通过，不把 GPIO 号当物理脚号。
```

### 可选:MCP 接入

仓库内的 [`mcp/`](mcp) 是一个本地 stdio MCP 适配层,方便 Codex 等支持 MCP 的
agent 直接发现并调用 `easyeda_*` 工具。它复用现有 `easyeda` CLI/daemon,不会绕过
typed action、审计、workflow gate 或官方 `eda.*` API;任意 JavaScript 的
`debug.exec_js` 域不会通过 MCP 暴露。

```bash
npm --prefix mcp ci --ignore-scripts
codex mcp add easyeda-agent \
  --env EASYEDA_BIN="$(command -v easyeda)" \
  -- node "$(pwd)/mcp/src/server.mjs"
```

重启 agent 客户端后即可使用。其他 MCP 客户端使用同一 stdio command/env 配置;
详细工具清单与开发验证见 [`mcp/README.md`](mcp/README.md)。

## 效果演示

### 门禁控制板示例：本地数据 → SCH Apply → 真实原理图

器件、完整物理引脚、稳定网络 ID 和 NC 保存在本地连接图中。Agent 先依据实测引脚几何
计算功能 Lib 内的器件位置、方向和连线，再组合页面，通过 `sch apply` 顺序写入
EasyEDA，并回读核对。原位号与器件身份保持不变。

**23 个器件 · 165 个物理引脚 · 28 个网络 · 2 页原理图**

#### 电源与 RF 主控

![门禁控制板示例的电源与 RF 主控页：外围电路按引脚方向连接，粉色虚线框标识功能模块](docs/images/access-control-power-rf.png)

电源、RF 主控和下载接口分别组成 Lib；外围电路按引脚方向连接，端口长短错落，
粉色虚线框与 0.2 inch 标题由数据计算。

#### 对讲与外设接口

![门禁控制板示例的对讲与外设接口页：功能模块按 Z 字阅读顺序排列，各框随内容独立收紧](docs/images/access-control-talk-interfaces.png)

从左上角按 Z 字阅读顺序排列，每框随内容独立收紧并保留最小内边距；同行顶齐，
下一行按本行最大高度推进。标题利用上方或下方空档，减少额外高度。

#### Apply 实际执行

![门禁控制板示例的实际 Apply 阶段捕捉，加速播放](docs/assets/access-control-sch-apply.gif)

动图使用电源与 RF 主控页实际 Apply 捕捉的 12 张关键阶段导图，加速播放；两张静图源自 EasyEDA 官方导图，展示名称已匿名化。
录制脚本与复现步骤见 [Apply 动图捕捉](docs/schematic-showcase.md)。
两页布局与连接检查均为 0 错误、0 警告；官方 DRC 仍有 3 WARN，严格门禁未通过，部分文字避让仍待完善。
完整验证范围见 [1.4 发布与验证](docs/release-1.4.md)。

### 历史 PCB 案例：ESP32-S3 四层板

> **完整实战案例:[一份需求文档 → AI 全自动画完 ESP32-S3 四层板](docs/showcase-esp32-mini.md)** ——
> 19 器件原理图 + 四层 PCB(GND 内电层/VCC 电源层/天线禁铜/四角 M3),
> `pcb drc` Connection/Clearance 双归零、`pcb check` 0、`layout-lint` 100/100,附原生截图与全流程复盘。

下面的历史录屏来自 ESP32 回归板的真实 EasyEDA 画布，展示 PCB 布局、板框和铺铜：

![AI 在 EasyEDA 中完成 PCB 布局、板框和铺铜](docs/assets/demo-pcb-layout.gif)

下面这块板由 agent 驱动完整 PCB 流程产出——**自动布局 → 板框贴合 → 规则感知布线 → 4 层电源平面 → 丝印碰撞避让**——并在真实 EasyEDA 画布上验证(DRC 31 → 3、No-Connection 归零):

<p align="center">
  <img src="docs/assets/demo-esp32-board.png" width="560" alt="ESP32-S3 成品板:4层电源平面 + 圆角板框 + 位号对齐" />
</p>

几个单步的真机前后对比(同一块板):

| `pcb outline-fit` 板框贴合(利用率 17% → 71%) | `pcb silk-align` 丝印碰撞避让 |
|---|---|
| <img src="docs/assets/demo-outline-before.png" width="330" alt="前:板框过大"/> → <img src="docs/assets/demo-outline-after.png" width="330" alt="后:板框贴合器件"/> | <img src="docs/assets/demo-silk-before.png" width="330" alt="前:位号散乱重叠"/> → 对齐后见上方成品板 |

> 本节 PCB GIF 和截图来自 ESP32 回归板真机流程(原理图 → 导入 PCB → 4 层叠层 → 布局 → GND 内电层/VCC 信号 plane → 天线禁区+检查 → 丝印/LED 极性 → 挖槽)。这是独立于门禁控制板示例的历史回归案例，原始需求见 [esp32MiniRequire.md](esp32MiniRequire.md)。

## 能力清单(已支持)

以 typed CLI 子命令暴露(`easyeda <domain> <verb>`)。1.4.5 的验证范围与未完成项见 [1.4 发布与验证](docs/release-1.4.md)。

**原理图** — 完整功能地图(已支持 40+ 子命令按功能域 + 待支持路线)见 **[docs/cli/schematic.md](docs/cli/schematic.md)**(CLI 功能索引:[docs/cli/](docs/cli/README.md));摘要:
- **器件与库**:从立创/LCSC 库按 uuid 放**真实器件**、换型号(`replace`)、符号/封装重绑、C 号确定性解析(`resolve-lcsc`);`modify` 属性 **merge 语义**(只 patch 顶层字段不再清空自定义属性,#175)。
- **连线**:`connect`/`autoconnect`(**打分器**自选方向——碰撞/穿件/图签/fanout 全几何成本,netport **竖排折叠惩罚**让密集引脚列标签保持水平)/`disconnect` 成对删;电源/地标志自动补偿旋转存储的坑。
- **数据与布局**:连接图保留稳定器件 ID、引脚、网络与 NC;合法数字位号保持原样,功能名称存 Role。`sch compose` 基于已设计的 Lib 几何离线组合单页,从左上向右按 Z 字排列；每框按内部内容独立收紧,同行顶齐,下一行按本行最大高度推进。`sch frame apply/check` 生成并检查保留最小内边距的粉色虚线框与 0.2 inch 标题。组合器不推导任意外围电路、不自动分页或删除源页。
- **转换与校验**:`sch apply` 串行执行规划队列,核对前置状态并回读引脚/网络/NC/几何;失败后重读重规划。`sch gate --strict` 四阶段(layout-lint→check→bridge-check→drc),覆盖重叠、悬空、短路与官方 DRC;`layout-score` 提供布局质量诊断。
- **跨页网名审计与对账**:`sch nets --strict`(网名变体/单引脚网机械拦截)+ `sch reconcile` 设计意图对账 + netlist **黄金表逐脚比对**——「接得合法」与「接对没有」分别有门。
- **电路块库**:`block-apply` 一键实例化验证过的拓扑(37 块:19 ready / 13 verified / 5 draft,离线可查),放件+连线+网表对账一条命令;`extract-layout` 真板反推模板。
- 一次调用 **`sch read`**(器件+网络+检查)、**BOM**/**网表**导出(自动补 LCSC C 号)、页面导图 SVG/PNG/PDF。

**PCB — 布局**
- **`pcb new-board`** — 从原理图**新建一块板 + 空 PCB 页**并绑定(CLI 版「新建 PCB / 原理图转 PCB」),再 `pcb import-changes` 从零布局;区别于只做链接的 `board.create`。
- **`pcb auto-place`** — 模块感知启发式:卫星器件贴到它所连芯片引脚那侧,2 脚器件自动转向,多芯片铺开;**间距规则感知**(由 live DRC clearance 推导),`--assembly-gap` 兜底手焊间距。
- **`pcb outline-fit`**(板框贴合器件)/ **`pcb outline-round`**(圆角矩形板框)。
- **`pcb layout-lint`** — 布局质量 + **可布性评分**(飞线 MST + 跨网交叉),布线前预测。
- **`pcb silk-align`** — 位号**位置感知**避让重排(v2):按局部空隙 + 板上位置 + 拥挤轴给每个位号的 4 个方向打分,**避开别人的焊盘/器件体/禁区/板框/其它标签**;挤死的报告出来而非压到焊盘上。
- **`pcb silk-add`** / **`pcb silk-set`** — 加**自由丝印字串**(板注 / LED 极性 `+`/`−` 标记,可配层/字号/线宽/旋转,JLCPCB 可读默认)+ 批量调整已有丝印,含 **`--align --ref` 对齐参考**(板注居中到板框、标签对齐器件边)。
- **`pcb add-component`** — 往已有 PCB 加单个器件并连接其焊盘网络(绕过失效的增量 `import_changes`)。

**PCB — 布线与铜**
- **`pcb route-short`** — 启发式短线布线:每网 MST、**规则感知线宽**(信号 vs 电源)、**障碍感知** L 朝向、**默认跳电源/地网**(它们该铺铜)。
- **`pcb pour`**(规则感知铜到板边内缩)/ **`pcb pour-fit`** / **`pcb via-stitch`** / **`pcb rip-up`**。
- **`pcb power-planes`** — 4 层电源分配:GND + 电源各占**专用内平面** + 每焊盘过孔缝合,铺铜后把 **GND 内层翻成 内电层/PLANE**(信号层铺铜→翻类型→重灌的验证配方,DRC 干净),匹配常见客户叠层 **GND=内电层 / VCC=信号层**(把回归板 DRC 31→0、No-Connection 归零)。
- **`pcb region`**(禁铺铜/天线净空)/ **`pcb fill`** / **`pcb slot`**(挖槽 / MULTI 层板挖空)。

**PCB — 叠层、规则、制造**
- **`pcb stackup`** — 设铜层数(2/4/6…/32)+ 内层类型(信号↔平面/内电层)。
- **全链路规则感知** — daemon 读板子 **live DRC 规则**(`pcb drc-rules`)并遵循;缺失时回退到权威 **JLCPCB fab 规则参考**(真实分板型导出)。**`pcb drc`** 跑检查。
- **`pcb export-dsn`**(Specctra DSN,给外部 Freerouting,带禁布区注入)/ **`pcb import-autoroute`** / **`pcb snapshot`**。

**基础设施**
- Typed action 协议(`--help` 自描述、`easyeda actions` 目录)+ `debug.exec_js` 原型逃生口。
- **`easyeda notify`** — 在 EasyEDA 窗口内弹**非阻塞 toast**(info/success/warn/error/question),流程可实时播报每一步(「完成布线,下一步铺铜」)。
- 连接器**自愈重连看门狗**(daemon 重启/窗口后台都能自动回来)+ daemon **防抖自动保存**。

## 暂不支持 / 平台墙

诚实说明边界。2026-07-01 对官方市场的扫描([docs/marketplace-coverage.md](docs/marketplace-coverage.md))校正了这些——真正的墙只在**交互式 UX** API,大多数「结果」(走线/过孔/泪滴/网长)其实够得到,进了吸收清单而非被堵死:

- **迷宫档自动布线**(密集/任意距离/推挤)—— daemon 只做*短、清晰*的启发式布线。完整布线走外部 **Freerouting**(DSN 往返构件已就绪);turnkey 集成**暂缓**(需 Java;等官方自动布线器过 `@alpha`)。
- **交互式布线 UX** —— 交互*菜单*(推挤拖拽布线、实时等长绕蛇、去环)**无 `eda.*` API**。但它们的*输出*——差分对几何、扇出打孔、等长绕线——可用 `pcb_PrimitiveLine/Via.create` 写出,所以**可作为我们的启发式实现**(吸收清单,非墙);只有拖拽 UX 是 UI 专属。
- **受控阻抗 Z0** —— 真的墙:叠层 Er / 介质厚 / 铜厚 `eda.*` 读不到,算不了 Z0 线宽。**但网长能读**(`pcb_Net.getNetLength`),所以等长/skew/时序余量报告可做(吸收清单)——这块之前被我误标成墙。
- **泪滴(teardrop)** —— 无*typed* create API;但文档源注入路径(如 `eext-balance-copper` 做 net-less 填充那样)可能可行,未验证。暂时 UI 里手动应用。
- **无编程 undo** —— `eda.*` 没有 undo/redo;回滚靠自建(数据快照 + 反向操作)。
- **增量 `import_changes`** —— 对 API 新增器件是 no-op(平台限制);首次同步前放完整电路,或用 `pcb add-component`。
- **丝印密度极限** —— `silk-align` 在有空白处避让标签;比标签更密的布局无法完全消重(报 `unresolvedCollisions`)——请放松布局。

市场覆盖矩阵 + 优先吸收清单见 [docs/marketplace-coverage.md](docs/marketplace-coverage.md);动作清单见 [docs/FEATURES.md](docs/FEATURES.md);`eda.*` API 覆盖地图见 [docs/ecosystem-survey.md](docs/ecosystem-survey.md)。

## 仓库结构

```text
cmd/easyeda/                 CLI 入口(人和 Skill 都用)
internal/app/                CLI 命令实现
internal/daemon/             本地 daemon:/health、/eda(连接器 WS)、/action
internal/protocol/           与连接器共享的 typed action 协议(actions.go)
extension/                   EasyEDA 连接器(.eext)源码 + 构建(TypeScript → esbuild)
skills/easyeda-agent/        合并后的公开 Skill:工作流、参考、脚本、规范数据
docs/                        架构、协议、功能/路线图、规范、决策
```

## 设计定位

裸 JavaScript 执行对调试仍有用,但不作为主要的 AI 界面。默认界面应该是**有类型的动作**:明确输入、可预测输出、产物处理、校验钩子。

延伸阅读:[快速开始 & 使用注意事项](docs/quick-start.md) · [功能清单与路线图](docs/FEATURES.md) · [架构](docs/architecture.md) · [协议](docs/protocol.md) · [Skill 设计](docs/skill-design.md) · [开发环境与调试手册](docs/dev-environment.md)

## 致谢

特别感谢 **嘉立创EDA(EasyEDA 专业版 / 嘉立创)** 开放的**扩展插件通道**和官方 `eda.*`
API。整个自动化层都建立在这个开放的插件平台之上——没有它,就没有这个项目。
`easyeda-agent` 始终做官方插件体系里一个薄而规矩的「公民」,这里的每一项能力最终都
落到嘉立创自己的 `eda.*` 调用上。感谢嘉立创让我们能做出这样一个好用的插件。

### 引用项目与前置工作(鸣谢)

站在这些开源项目的肩膀上——感谢:

- [**@jlceda/pro-api-types**](https://www.npmjs.com/package/@jlceda/pro-api-types) —— 官方 EasyEDA Pro `eda.*` API 类型定义(连接器对它做类型校验)。
- [**Freerouting**](https://github.com/freerouting/freerouting) —— 外部迷宫档自动布线器,我们的 `pcb export-dsn` / `import-autoroute` 往返对接它。
- [**spf13/cobra**](https://github.com/spf13/cobra)(CLI 框架)· [**coder/websocket**](https://github.com/coder/websocket)(daemon ↔ 连接器)· [**esbuild**](https://github.com/evanw/esbuild)(连接器打包)。
- **官方 EasyEDA 扩展**([github.com/easyeda](https://github.com/easyeda))—— 我们研究它们的 `eda.*` API 用法与算法(不抄 UI)作为前置工作;吸收清单见 [`docs/ecosystem-survey.md`](docs/ecosystem-survey.md)。其中 [`eext-run-api-gateway`](https://github.com/easyeda/eext-run-api-gateway) 证明了编辑器内代码通道,[`eext-export-design-report`](https://github.com/easyeda/eext-export-design-report) 启发了设计报告读取。
- 尚未吸收的候选:[**polyclip-ts**](https://github.com/luizbarboza/polyclip-ts)(多边形布尔)—— 用于未来的丝印填充避让(见 `docs/ecosystem-survey.md` A10)。

## 许可证

[MIT](LICENSE) —— 随便用、随便改、随便商用,保留版权声明即可。

唯一例外:`extension/src/beautify/` 下的四个文件移植自
[Easy_EDA_PCB_Beautify](https://github.com/m-RNA/Easy_EDA_PCB_Beautify)(作者 m-RNA),
沿用其 **Apache-2.0** 许可(许可证全文在
[`extension/src/beautify/LICENSE`](extension/src/beautify/LICENSE),署名与改动清单见
[`NOTICE`](NOTICE))。两者兼容,不影响整体以 MIT 使用。

## Star History

感谢每一颗 star。

[![Star History Chart](https://api.star-history.com/svg?repos=zhoushoujianwork/easyeda-agent&type=Date)](https://www.star-history.com/#zhoushoujianwork/easyeda-agent&Date)
