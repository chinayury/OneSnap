# 架构说明

OneSnap 是一个 Wails v2 桌面应用：**Go 原生后端 + React WebView 前端**，
两者通过 Wails 绑定（方法调用）与事件总线通信。

## 运行时结构

```
┌───────────────────────────── Windows 进程 ─────────────────────────────┐
│                                                                        │
│  Go 后端（单实例）                    WebView2（React 编辑器）          │
│  ┌──────────────────────┐            ┌──────────────────────────┐      │
│  │ hotkeys  全局热键     │  Events    │ App.tsx  状态与捕获流程   │      │
│  │ overlay  原生选区层   │ ─────────► │ EditorCanvas (Konva)     │      │
│  │ screenshot 捕获/剪贴板│ ◄───────── │ AnnotationShapes 标注    │      │
│  │ tray     托盘         │  Bind 调用 │ Toolbars / Settings      │      │
│  │ config   配置/自启    │            │ Library / UpdateModal    │      │
│  │ library  截图库       │            └──────────────────────────┘      │
│  │ updater  更新检查     │                                              │
│  └──────────────────────┘                                              │
└────────────────────────────────────────────────────────────────────────┘
```

## 核心流程：区域截图

1. 热键触发 → Go 发出 `hotkey:region` 事件 → 前端调用 `PrepareRegionCapture`；
2. Go 隐藏主窗口，捕获整个虚拟桌面原始 RGBA（不编码 PNG，省时间）；
3. `overlay.Manager` 在独立线程创建 Win32 分层窗口（`WS_EX_LAYERED` +
   `UpdateLayeredWindow`），把截图绘制为背景并叠加 50% 暗化遮罩；
4. 用户拖选（支持空格移动），选区经 `scaleRatio` 换算为物理像素后直接
   在 Go 侧裁剪并编码 PNG，经 `region:selected` 事件回传前端；
5. 前端进入编辑器，自动复制（若启用）。

选区层提示文字用 GDI 真字体渲染（Microsoft YaHei UI，按 DPI 缩放）：
GDI 会破坏 32bpp DIB 的 alpha 通道，所以先把字形渲染到独立小 DIB，
再按灰度覆盖率混合回主像素缓冲（预乘 alpha）。

## 编辑器与标注

- 所有标注是不可变数据（`Annotation` 数组），通过 undo/redo 快照栈管理；
- 形状渲染在 `annotation-shapes.tsx`，选区/缩放用 Konva Transformer；
- **马赛克/模糊**：对整张截图预生成像素化/模糊画布（display 尺寸坐标），
  标注形状以 `clipFunc` 裁剪显示，拖动时实时同步保证像素对位；
- 裁剪是快照式：保留原图与标注，可反复重裁、还原；
- 导出走 `stage.toDataURL/toCanvas`（隐藏 Transformer，1:1 像素比）。

## 配置与存储

- 配置文件：`%APPDATA%\OneSnap\config.json`（JSON，写盘前合并，
  设置弹窗只管理热键/启动/快存/导出/更新检查五组，避免误清其它设置）；
- 快速保存默认目录：`图片\OneSnap`，命名模式 timestamp/date/increment。手动与热键保存共用独占创建逻辑：时间戳和日期重名时追加数字后缀，递增模式使用下一个可用编号；
- 开机自启：`HKCU\...\CurrentVersion\Run`，每次启动若启用则同步 exe 路径；
- 更新检查：GitHub Releases 最新版 semver 比较，可跳过指定版本。

### 截图后动作的取舍

[WinShot](https://github.com/mrgoonie/winshot) 的标注和导出流程已由编辑器覆盖。托盘沿用其显示版本号的做法，直接展示构建时注入的版本字符串。[Flameshot](https://github.com/flameshot-org/flameshot) 支持直接保存和复制，OneSnap 的静默保存和编辑器导出共用保存逻辑，两处均需保留同名旧图。[ShareX](https://github.com/ShareX/ShareX) 支持上传到外部服务，OneSnap 的快速保存保持本地写盘，无需配置账户。

[ksnip](https://github.com/ksnip/ksnip) 支持的像素化和序号命名已由编辑器和快速保存覆盖。[Greenshot](https://github.com/greenshot/greenshot/blob/main/docs/capture-user-guide.md) 和 [ShareX](https://github.com/ShareX/sharex.github.io/blob/master/docs/region-capture.md) 支持复用上次选区。OneSnap 暂缓此项，跨屏复用前需确定显示器断开或 DPI 改变时如何恢复选区。

编辑器在另存为成功后记录后端返回的完整路径，供「复制路径」使用。取消或写盘失败会保留上次成功保存的路径。另存为和快速保存开始时都会记录当前图片的代次，以免异步写盘完成时覆盖新打开图片的路径。[Flameshot 保存实现](https://github.com/flameshot-org/flameshot/blob/master/src/utils/screenshotsaver.cpp)也在写盘成功后使用最终路径。[ShareX](https://getsharex.com/) 将另存为和复制路径列为独立动作，OneSnap 同样由用户决定何时复制另存为的路径。

编辑器快捷键只在主界面生效。打开截图库、设置、窗口选择或更新对话框时，标注、导入导出和画布快捷键暂停，关闭后恢复。例如，在截图库按 Delete 只会询问是否删除选中的截图。[Flameshot 快捷键说明](https://flameshot.org/docs/guide/key-bindings/)也按操作界面区分快捷键的用途。

### 截图库的加载上限

打开截图库时，OneSnap 先按文件修改时间从新到旧排序，再依次解码缩略图，最多显示 500 张。修改时间相同的图片按文件名排序。遇到损坏或无法读取的图片会继续向后查找，直到填满列表或扫描结束。[ShareX 的图库更新记录](https://getsharex.com/changelog-classic)也介绍了通过限制图片数量来缩短大型图库加载时间的做法。

## 构建版本号

版本号单一来源为根目录 `package.json` 的 `version` 字段：

- 前端经 `vite.config.ts` 的 `__APP_VERSION__` 注入（界面显示、更新检查）；
- 后端由 CI 以 `wails build -ldflags "-X main.Version=v<version>"` 注入；
- 发布 tag 必须与 `package.json` 版本一致（release 工作流强制校验）。
