# Release 治理规范

本仓库的版本发布遵循以下规范，确保 release 资产完整、可追溯、可复现。

## 发布流程

1. **打 tag**：`git tag v<major>.<minor>.<patch>` 并推送
2. **触发构建**：推送 tag 后 GitHub Actions（`.github/workflows/build-release.yml`）自动构建并创建 release
3. **人工验证**（构建完成后的检查清单）：

## 发布前检查清单

- [ ] 新版本号已更新（`internal/common/version` 或构建参数）
- [ ] `go build ./...` 本地编译通过
- [ ] CHANGELOG（或 release notes）记录了本版变更
- [ ] tag 与 go.mod / workflow 中的 module 路径一致（`github.com/wwwangzilin/LotsACG-Standalone`）

## 发布后验证清单

- [ ] Release 已创建，名称与 tag 一致
- [ ] 资产文件全部上传且**大小正常**（对比上一版，差 >5% 视为异常）
- [ ] 资产命名符合规范（见下）
- [ ] 最新 release 附带 `SHA256SUMS` 校验文件

## 资产命名规范

统一使用 `LotsACG_<版本>_<平台>.<ext>`：

| 平台 | 命名 |
| --- | --- |
| Windows | `LotsACG_<版本>_windows.exe` / `.zip` |
| Linux | `LotsACG_<版本>_linux` / `.zip` |
| macOS | `LotsACG_<版本>_darwin` / `.zip` |

> 避免使用 `LotsACG_new.exe` 这类不含版本信息的命名——无法区分版本，也无法校验是否对应某 tag。

## 生成 SHA256SUMS

```sh
# 在资产所在目录执行
sha256sum LotsACG_* > SHA256SUMS
# Windows PowerShell:
Get-FileHash LotsACG_* -Algorithm SHA256 | ForEach-Object { "{0}  {1}" -f $_.Hash, $_.Path.Split('\')[-1] } | Out-File SHA256SUMS -Encoding ascii
```

## 资产完整性校验

```sh
# 校验下载的资产
sha256sum -c SHA256SUMS
```

## 与上游的关系

- 本仓库是 `krau/ManyACG` 的独立维护版本（AGPL-3.0）
- 上游更新**不会**自动同步到本仓库；如需合并上游，手动 cherry-pick 或 merge 并重新验证
- 每次发布都从本仓库 `v1` 分支构建，不依赖上游状态
