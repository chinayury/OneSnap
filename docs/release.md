# 发布流程

## 触发方式

推送 `vX.Y.Z` 格式的 tag 自动触发 `.github/workflows/release.yml`。

## 版本规则

- 版本号**唯一来源**：根目录 `package.json` 的 `version` 字段；
- 打 tag 前确认 `package.json` 已同步（工作流会校验 tag 与版本一致，不一致直接失败）；
- 常规修复 → patch；新增用户可见功能 → minor；不兼容变更 → major。

## 工作流步骤

1. Windows runner 检出代码，安装 Go 1.26 / Node 22 / Wails CLI v2.17.1；
2. 校验 tag 与 `package.json` 版本一致；
3. `wails build -ldflags "-X main.Version=v<version>" -clean` 产出 `build/bin/OneSnap.exe`；
4. 打包 `OneSnap-<version>-Windows-x64.zip`；
5. 删除所有历史 Release 附件（保留 Release 记录本身）；
6. 上传当前附件并按 tag 生成 Release Notes。

## 发布后检查

- [ ] Release 附件只有当前版本
- [ ] CI（若同 tag 提交触发）成功
- [ ] 更新检查可用：启动旧版应用应提示新版本
- [ ] README 不包含具体版本号（产物名用 `<version>` 占位）
- [ ] 远端无 token / 密码 / 本机绝对路径

## 本地验证清单（发布前）

```bash
go build ./...
go vet -unsafeptr=false ./...
go test ./...
cd frontend && npm ci && npm run build
wails build
```

产物可启动、托盘出现、三种截图可用、编辑器标注与导出正常。
