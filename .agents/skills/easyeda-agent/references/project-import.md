# 外部工程导入与迁移

本页分别描述 Altium Designer 工程迁移的能力边界，以及原生 `.epro2` 工程的打开、导出和恢复。

## Altium Designer 工程：当前 `unsupported`

当前没有经过验证的 typed action 或 CLI 能把 `.SchDoc` / `.PcbDoc` 程序化导入
EasyEDA Pro。Agent 不得用交互界面、CUA、`debug.exec_js`、分块 base64 上传或私有消息总线
兜底。需要导入时停止现场写入，将能力标为 `unsupported`，先实现并验证 typed import。

如果目标工程已经由用户独立提供，easyeda-agent 只从现有工程开始读取与核验：

1. `easyeda health --project <project>` 与 `easyeda doc ls --project <project>`：确认连接器、
   目标工程，以及导入所得的原理图/PCB 文档。
2. 对每张原理图读取 `easyeda sch connectivity --page <name-or-uuid> --project <project>`，
   核对器件身份、全部物理引脚、网络、NC、分层页面与位号；再逐页运行适用的
   `layout-lint`、`sch check`、`bridge-check` 和 SDK DRC，分别保存结果。
3. 对 PCB 核对 Board 绑定、器件与焊盘网络、板框、层叠、机械层/禁布区、关键网和
   丝印；运行 `easyeda pcb layout-lint`、`easyeda pcb check --strict` 与
   `easyeda pcb drc --strict`。AD 机械层到 EasyEDA
   层的映射必须在原生画布和数据回读中确认，不能只看工程已出现在目录里。
4. 记录导入前后的数量与关键不变量。缺页、空读、板框/机械层变化、网络不一致或检查
   未运行时，均不能报告迁移成功；修复完成后显式保存。

## 为什么不能调用名义上的导入 API

官方 beta 方法 `eda.sys_FileManager.importProjectByProjectFile` 的签名列出了
`'Altium Designer'` / `'Protel'` 类型，但在 issue #203 的 EasyEDA Pro 桌面端
3.2.149（Windows x64，本地工作区）实测中，对 `.epro2`、`.eprj2` 和真实 `.SchDoc`
都会快速 resolve 为 `undefined`，不抛错，也没有创建文档或改变工程。

因此：

- 方法存在、Promise resolve 或返回 `undefined` 均不算成功；不得继续后续自动化。
- `getAllProjectsUuid()` 在本地工作区可能返回 `[]`，不能单独作为导入副作用判据。
- `eda.sys_FormatConversion.convertAltiumDesignerLibrariesToEasyEDA*` 只转换
  `.SchLib` / `.PcbLib` 库文件，不转换 `.SchDoc` / `.PcbDoc` 工程文档。
- 当前不封装 `import.ad` typed action。将来只有在受支持宿主上证明真实副作用，并能在
  返回值缺失、超时和部分导入时明确失败，才可进入 CLI/action 实现。

未来接口封装至少要在写入前记录目标工程指纹，并在返回后核对工程/文档身份、文档清单、
原理图连接数据及 PCB 板框/机械层；不得把 `undefined`、无变化或单一 UUID 清单当成功。

来源与完整探测矩阵见 [GitHub issue #203](https://github.com/zhoushoujianwork/easyeda-agent/issues/203)。


## 工程级打开与原生导出

`project open --uuid` 是旧的文档打开别名，只能用于当前工程内页面，不是跨工程打开。
跨工程时先保存所有未保存文档，再显式调用：

```bash
easyeda project open --window <window-id> --project-uuid <project-uuid> --page-uuid <page-uuid> --allow-discard-unsaved
easyeda project export --window <window-id> --project-uuid <project-uuid> --out ./deliverable.epro2
```

打开命令使用官方 `dmt_Project.openProject`；该 API 可能丢弃未保存数据，因此标志是明确确认，
不是自动保存。命令核对打开后的真实工程身份；超时或失败先读回，不盲目重复。
导出使用官方 `sys_FileManager.getProjectFile`，要求目标工程已激活，前后检查 UUID，
拒绝覆盖文件，校验原生 ZIP 完整性并输出字节数/SHA-256。输出 `restoreVerified=false`：
ZIP 校验和导出成功不代表重新导入验证通过。请在导出前显式保存所有文档。
底层调用正式 typed action `project.open` / `project.export`，需要注册这两个 action 的新版 daemon 及包含 handler 的新版连接器。旧 daemon 或连接器拒绝 unknown action / UNKNOWN_ACTION 时停止并升级；禁止回退到 debug.exec_js。
直接调用 `project.open` 同样必须传 `allowDiscardUnsaved:true`；`project.export` 不接受页面路由。
限制：归档最大 16 MiB，解压验证上限 128 MiB，超限明确失败。

MCP 使用 `easyeda_project_transfer`，`operation` 为 `open` 或 `export`，均需 `window`、
`projectUuid`；打开另需 `allowDiscardUnsaved=true`，导出另需新 `out` 路径。不传文档路由。
此工具需要含上述 CLI 命令的匹配构建；不要仅替换 MCP 而仍使用旧 CLI。

工程身份可能先于文档树就绪。需直接进入原理图时，打开命令同时传 `--page-uuid`（MCP `pageUuid`），等待目标页面出现在树中后只打开一次，并核对工程和页面身份。省略此参数只保证工程身份，不保证页面已加载。

## 原生工程恢复：New Project typed import

`.epro2` 导出与 ZIP 校验不能证明能在新工程恢复。官方 beta `sys_FileManager.importProjectByProjectFile` 提供 EasyEDA Pro/JLCEDA Pro 与 New Project 参数，开发版已实现下述 typed import。2026-09-22 在 Windows、EasyEDA Pro 3.2.149.88089769、ONLINE、匹配 1.5.3-dev.5 CLI/daemon/connector 上，正常及断线原理图包均完成新工程导入、独立回读、保存重载及源工程对照；断线状态保持。覆盖限于单页原理图，不证明 AD 迁移、离线模式、多页层级或 PCB 恢复已验证。上述本地工作区 AD 探测也不能推出所有宿主模式下的原生导入均不支持。

要求可恢复交付时，每次任务均应仅导入到明确的新目标工程，验证源文件哈希与归档边界、源工程不变、新身份和文档清单，并按位号/真实引脚对照参数、库身份及网络集合。保存重载后再次对照；目标 UUID 非空或导入 Promise resolve 均不能单独证明恢复成功。未完成本次对照就保留 NOT-VERIFIED/incomplete，不以历史成功替代本次验收。

接口来源：https://prodocs.lceda.cn/cn/api/reference/pro-api.sys_filemanager.importprojectbyprojectfile.html （beta，参数须按实际宿主核对）。

开发接口（需匹配含 handler 的新连接器和 daemon）：

```bash
easyeda project import --window <window> --project-uuid <active-source> --file ./source.epro2 --team <owner-uuid> --name <unique-new-name> --allow-discard-unsaved
```

MCP `easyeda_project_transfer(operation=import, window, projectUuid, file, teamUuid, friendlyName, allowDiscardUnsaved=true)`；不传 base64，不走通用 action 的长命令路径。仅 New Project/ImportDocument，不覆盖原工程或主动提取库。先保存全部文档，确认当前源身份和目标 owner；文件上限 16 MiB、解压总量 128 MiB、最多 2048 条目，检查路径、CRC 和 SHA-256。

成功返回仅表示新工程身份/owner/name 核验，restoreVerified 和 sourceUnchangedVerified 仍为 false，必须独立保存重载及内容对照后才能提升结论。相同源/owner/name/hash 的调用在同一连接器进程内只执行一次，失败也缓存；重启后不保证去重，超时/undefined/部分成功先查新工程，不盲目重试。枚举到同名工程时拒绝导入；这不替代跨进程事务。

独立对照保留原始响应，不改写工具返回的 false。将经验证的恢复结论另写入报告，引用源归档 SHA-256、源/新工程身份、页面清单、导入前后及重载后的对象/网络证据。故障包应保留已知故障，不能把“导入后检查失败”直接等同恢复失败。原生导出可改变文档块排序和 DOCHEAD.client；字节哈希不同先定位差异，若使用规范化比较，必须记录精确排除字段并保留原件，不能宽泛忽略属性变化。
