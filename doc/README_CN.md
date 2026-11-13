# Lumina Amber（光珀）

> **作者**：`eddy` · **当前版本**：`0.9.0` · **开源协议**：GNU GPL v3.0  
> 像琥珀封存光阴一样，离线整理、去重并留存每一帧珍贵照片与视频（Go + Fyne v2，支持 Linux & macOS）。

[English README](../README.md)

---

## 核心特性

- **100% 离线与零隐私泄露**：纯本地运行，零联网、零统计、零广告。
- **双模式一体化**：
  - **手机备份**：支持 Android **ADB（USB / 无线调试）** 及 **挂载目录 / Linux GVFS MTP**，具备远程 `sha256sum` 预检去重能力。
  - **目录整理**：递归导入本地或移动硬盘目录，按统一规范拷贝或校验后移动归档。
- **照片与视频分目录按年月归档**：
  - 照片：`<目标目录>/picture/<YYYY>/<YYYY-MM>/<文件名>`
  - 视频：`<目标目录>/video/<YYYY>/<YYYY-MM>/<文件名>`
  - 时间优先级：EXIF `DateTimeOriginal` -> MP4/MOV `mvhd` 创建时间 -> 合法文件名日期 -> 文件修改时间。
- **目标目录便携索引（`<目标目录>/meta/index.db`）**：
  - 所有内容哈希（SHA-256）与同步历史全部存放在目标备份盘自身的 `meta/` 目录内，更换目标盘或插到其他电脑即可无缝继续增量去重备份。
- **高并发批处理流水线**：
  - 多核并行计算目标盘与本地目录哈希、`sync.Pool` 1 MiB 共享缓冲区、单次读取同步计算 SHA-256、内存快照查重以及 64 条批量 SQLite 事务落盘。
- **现代三主题桌面体验**：提供 **macOS 浅色**、**macOS 深色** 与 **AMOLED 纯黑** 三种主题，自动记忆上次设备、目标盘与筛选条件，打开即可一键备份。

---

## 本地构建

### Linux
```bash
# 需提前安装 GCC 与 OpenGL/X11 开发头文件（libgl1-mesa-dev xorg-dev libxxf86vm-dev）
make test
make build-linux
./bin/lumina-amber-linux-amd64
```

### macOS
在 macOS 本机或通过 GitHub Actions（`macos-latest`）可直接打包通用架构 `.app`：
```bash
CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/macos/lumina-amber ./cmd/lumina-amber
./scripts/package-mac.sh bin/macos/lumina-amber
```

推送 `v*` 标签（例如 `git tag v0.9.0 && git push origin v0.9.0`）将由 GitHub Actions 自动构建 Linux 与 macOS 通用版安装包并发布到 **GitHub Releases**。
