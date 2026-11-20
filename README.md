# OneSnap

[![CI](https://github.com/chinayury/OneSnap/actions/workflows/ci.yml/badge.svg)](https://github.com/chinayury/OneSnap/actions/workflows/ci.yml)

一键截图 · 标注 · 保存工具（Windows 桌面应用）。

按下快捷键，框选屏幕任意区域，立刻进入编辑器：加箭头、写文字、打序号、
打马赛克/模糊，配上好看的渐变背景，一键复制或保存。

基于 [Wails](https://wails.io) 构建：Go 原生后端 + React 画布编辑器，
安装包体积小、启动快、无运行时依赖（Windows 10/11 自带 WebView2）。

## 功能特性

**截图**

- 区域截图：原生 Win32 覆盖层，跨屏拖选，`空格` 移动选区，`Esc` 取消
- 全屏截图 / 窗口截图（带实时缩略图选择器）
- 剪贴板导入、本地图片导入、拖拽导入
- 多显示器与高 DPI（per-monitor DPI aware）支持

**标注编辑器**

- 矩形 / 椭圆 / 箭头（直线与曲线）/ 直线 / 文字 / 步骤序号 / 聚光灯
- **马赛克 / 模糊**：一键遮盖敏感信息（OneSnap 新增）
- 裁剪（自由比例与 16:9 / 4:3 / 1:1 / 9:16 / 3:4 预设）
- 样式化背景：渐变预设、圆角、阴影、边框、自动取色
- 撤销 / 重做（Ctrl+Z / Ctrl+Shift+Z）

**导出**

- PNG / JPEG（JPEG 质量可调），带样式的整图导出
- 另存为、快速保存（自动命名：时间戳 / 日期 / 递增序号）、复制到剪贴板
- 截图库：浏览、编辑、删除历史截图
- 捕获后自动复制（可开关）

**系统集成**

- 全局热键（默认 `PrintScreen` 全屏、`Ctrl+PrintScreen` 区域、`Ctrl+Shift+PrintScreen` 窗口，均可自定义）
- 系统托盘：左键打开截图库，双击显示主窗口，右键菜单（跟随界面语言）
- 开机自启、关闭到托盘、最小化到托盘
- 启动时检查更新（GitHub Releases）

**双语界面**

- 中文 / English 一键切换：设置 → 通用，或托盘菜单「切换语言 / Switch Language」，即时生效并持久化
- 全部按钮带悬停备注说明（随界面语言同步）
- 选区遮罩操作提示同步切换中 / 英文

## 快捷键

| 操作 | 默认热键 | 可自定义 |
| --- | --- | --- |
| 全屏截图 | `PrintScreen` | ✓ |
| 区域截图 | `Ctrl+PrintScreen` | ✓ |
| 窗口截图 | `Ctrl+Shift+PrintScreen` | ✓ |
| 另存为 | `Ctrl+Shift+S` | 固定 |
| 快速保存 | `Ctrl+S` | 固定 |
| 复制到剪贴板 | `Ctrl+C` | 固定 |
| 粘贴剪贴板图片 | `Ctrl+V` | 固定 |
| 导入图片 | `Ctrl+O` | 固定 |
| 标注工具切换 | `V` `R` `E` `A` `L` `T` `N` `S` `M` `B` `C` | 固定 |

## 安装

1. 打开 [Releases](https://github.com/chinayury/OneSnap/releases/latest) 页面；
2. 下载 `OneSnap-<version>-Windows-x64.zip` 并解压；
3. 运行 `OneSnap.exe`（首次运行 Windows SmartScreen 可能提示，选择「仍要运行」）；
4. 程序启动后驻留托盘，直接按全局热键截图。

系统要求：Windows 10 (1809+) 或 Windows 11，x64。

## 从源码构建

依赖：Go 1.24+、Node.js 18+、[Wails CLI v2](https://wails.io/docs/gettingstarted/installation)、WebView2 Runtime（Win11 自带）。

```bash
git clone https://github.com/chinayury/OneSnap.git
cd OneSnap
wails build
```

产物位于 `build/bin/OneSnap.exe`。

开发模式（热重载）：

```bash
wails dev
```

## 项目结构

```
├── main.go                  # 入口：单实例互斥锁、Wails 启动配置
├── app.go                   # 绑定到前端的全部后端 API
├── internal/
│   ├── config/              # 配置读写、开机自启（注册表）
│   ├── hotkeys/             # 全局热键（RegisterHotKey 消息循环）
│   ├── overlay/             # 原生选区覆盖层（分层窗口 + GDI，中文提示）
│   ├── screenshot/          # 屏幕捕获、窗口捕获、剪贴板图片读取
│   ├── windows/             # 窗口枚举与缩略图
│   ├── library/             # 截图库扫描与缩略图
│   ├── tray/                # 系统托盘（中文菜单）
│   └── updater/             # GitHub Releases 更新检查
├── frontend/                # React + TypeScript + Konva 编辑器
│   └── src/components/      # 工具栏、画布、标注形状、设置、库窗口
└── build/                   # 图标与 Windows 构建资源
```

## 与上游的差异

OneSnap 基于 WinShot 项目深度改版而来（感谢 [mrgoonie/winshot](https://github.com/mrgoonie/winshot)）。
主要变化：

- 新增**马赛克 / 模糊**标注工具
- 选区覆盖层提示与托盘菜单**中文化**（GDI 真字体渲染，支持 DPI 缩放）
- 移除云上传（Cloudflare R2 / Google Drive）及 AWS/Google SDK 重依赖，聚焦本地场景
- 修复设置保存会清空编辑器偏好/窗口尺寸等无关配置的问题
- 修复剪贴板 DIB 解析中的边界检查，并按 Go vet 规范化 unsafe 指针用法
- 全新图标、文档与构建/发布流水线

## 路线图

- [ ] 贴图（Pin to Screen）：截图钉在桌面置顶显示
- [ ] 长截图（滚动拼接）
- [ ] GIF / MP4 录制
- [ ] 多语言界面

## 许可证

本项目以 [GPL-3.0-or-later](LICENSE) 发布。项目包含源自 WinShot（BSD-3-Clause）的代码，
相关版权与许可声明见 LICENSE 文末附注。
