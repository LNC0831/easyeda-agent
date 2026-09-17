# 数据手册驱动的自动建库

目标是让用户只提供准确型号或 PDF：Agent 通读手册、提取带页码的证据、生成确定性几何，
再由 CLI 校验并写入 EasyEDA。不要让用户重复选择手册中已经明确的参数；只有封装后缀、
多种 land pattern 或手册缺失等真实歧义才询问用户。

## 工作流

1. 先 `lib search` / `lib by-lcsc` 查现有 Device。型号、封装和引脚定义完全匹配时优先复用，
   不重复建库。
2. 获取原厂 PDF 并通读目录、订购信息、pin table、package drawing、recommended land pattern。
   扫描件先 OCR；表格与尺寸图同时看文字层和页面图。必须锁定完整 MPN 与封装后缀，不能把
   同系列其他 variant 的引脚或尺寸混入。
3. 写一份完整 Device JSON。`evidence` 记录厂家、MPN、封装变体、手册定位信息，以及引脚表、
   外形图、推荐焊盘各自的 1-based PDF 页码。可保存 SHA-256，防止同 URL 文档静默换版。
4. Symbol 使用真实 physical pin number。可按功能分组排布，名称和电气类型来自手册；不因
   多个 GND 同名而合并编号。Footprint 单位为 mil，尺寸换算固定为 `1 mm = 39.37007874 mil`。
5. Land pattern 的依据只能是：`datasheet`（厂家推荐图）、`calculated`（按明确标准/公式计算，
   在 note 写标准、目标密度和关键假设）、`copied-verified`（复制已有封装并在 note 写来源与
   尺寸对账）。器件 body/lead 尺寸不能直接冒充铜焊盘尺寸。
6. 运行 `easyeda lib device validate --spec device.json`。默认要求 symbol pin 与 footprint pad
   编号集合完全相等；机械焊盘等例外必须在 `pinMapping.footprintOnly` / `symbolOnly` 逐项声明。
7. 门禁通过后运行 `easyeda lib device build --spec device.json`。该命令在任何 EasyEDA 写入前
   重跑同一校验，然后创建 Symbol、Footprint、可选 3D Model 和 Device；失败会尝试回滚。
8. 回读 Device 绑定、symbol pins 与 footprint pads，放置一个实例核对 Pin-1/极性/旋转，运行
   原理图检查和 PCB DRC 并保存。`verified:true` 只证明对象成功创建，不证明封装电气/机械正确。

## 规格契约

从 [library-device-example.json](library-device-example.json) 复制骨架。关键字段：

- `evidence.datasheet.locator` 可为原厂 URL 或本地 PDF 路径；`pinoutPages`、
  `packageDrawingPages` 至少各一页。
- `landPattern.basis=datasheet` 必须给 `pages`；`calculated` / `copied-verified` 必须给 `note`。
- `symbol.geometry` 直接传给 `library.symbol.build`：`outline[]` 至少四个 x/y 点，`pins[]` 的
  `number/name/x/y` 必填。
- `footprint.geometry` 直接传给 `library.footprint.build`：每个 pad 的 `number/layer/x/y/shape`
  必填，`shape` 为官方 pad tuple；图形线为 `lines[]`。
- 重复 pin/pad number、非有限坐标、非正焊盘尺寸、缺页码证据或未声明的映射差异都会在
  离线阶段拒绝，不打开 EasyEDA。

## 必须暂停的歧义

完整 MPN 对应多个封装、订购后缀无法确定、pin table 与 package 图冲突、推荐焊盘不存在且
计算依据不明确、EP/NC/DNU 的电气处理互相矛盾时，列出候选与证据页再询问用户。图像模糊时
换原厂 PDF 或更高分辨率页面，不凭轮廓猜尺寸。

官方 `eext-ai-library-builder` 的可吸收点是 PDF 转页图、自动定位 pin/package 区域和按封装类型
提取参数；其默认尺寸、一次视觉结果和人工框选都不能作为我们的最终真值。Agent 负责语义
理解，CLI 负责确定性校验和写入。
