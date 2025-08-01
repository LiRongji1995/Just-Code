# CLI 向 GUI 应用演进的架构设计指南

当你在开发 CLI 工具的初期就考虑到未来要演进成 GUI 应用，合理的架构设计将极大减少重构成本，并提升代码的复用性和可维护性。

------

## ✅ 总体原则：UI 是壳，逻辑是核

将 CLI 的输入输出、命令解析与核心逻辑解耦，使 GUI 可以无缝复用业务功能。

------

## 🧠 架构设计准则与建议

### 1. 分层架构（Clean Architecture）

| 层级           | 作用                                       | CLI 模式            | GUI 模式            |
| -------------- | ------------------------------------------ | ------------------- | ------------------- |
| **UI 层**      | 接收用户输入，显示状态                     | Cobra 命令行        | Fyne/Wails 页面     |
| **应用服务层** | 调用核心逻辑、参数验证、状态协调           | 封装函数调用        | 复用同一套 API      |
| **核心逻辑层** | 实现下载、做种、校验等核心功能逻辑         | 封装模块            | 封装模块            |
| **基础设施层** | 文件系统、网络、存储、数据库等系统调用封装 | 封装 I/O 等底层逻辑 | 封装 I/O 等底层逻辑 |



------

### 2. 项目结构建议

```
bash复制编辑myapp/
├── cmd/             # CLI 入口
│   ├── download.go
│   └── seed.go
├── gui/             # GUI 入口
│   └── main.go
├── app/             # 应用服务层：业务协调与接口暴露
│   ├── download.go
│   └── seed.go
├── core/            # 核心模块：下载器、元数据、校验器
│   ├── downloader/
│   ├── meta/
│   └── seeder/
├── infra/           # 系统层封装：文件 I/O、网络
│   └── storage.go
└── config/          # 配置管理
```

------

### 3. 开发建议

| 项目方面       | 建议说明                                                     |
| -------------- | ------------------------------------------------------------ |
| ✅ 核心逻辑封装 | 所有业务逻辑抽象为函数，例如 `StartDownload(metaPath string)` |
| ✅ CLI 输出解耦 | CLI 中避免直接使用 `fmt.Println()` 输出，改用日志或事件回调  |
| ✅ 支持异步任务 | GUI 是事件驱动的，逻辑应支持 goroutine、context、channel 等机制 |
| ❌ 不耦合 cobra | 不要把业务逻辑写死在 `cobra.Command.Run` 中，改为调用 `app` 层函数 |



------

## 📚 推荐资料

| 类型       | 名称                                                         | 内容说明                          |
| ---------- | ------------------------------------------------------------ | --------------------------------- |
| ✅ 模板项目 | [go-clean-arch](https://github.com/bxcodec/go-clean-arch)    | Go 的 Clean Architecture 项目模板 |
| ✅ GUI 模板 | [Wails Clean Template](https://github.com/GreenLightning/wails-clean-template) | GUI + Clean 架构                  |
| 📘 书籍     | 《Clean Architecture》 by Uncle Bob                          | 架构理念经典著作                  |
| 📘 书籍     | 《The Go Programming Language》                              | Go 官方权威教材                   |



------

## 🧩 开源项目参考

| 项目名称                                                     | 类型      | 特点说明               |
| ------------------------------------------------------------ | --------- | ---------------------- |
| [`task`](https://github.com/go-task/task)                    | CLI 工具  | 结构清晰，适合包装 GUI |
| [`aria2`](https://github.com/aria2/aria2) + [WebUI](https://github.com/ziahamza/webui-aria2) | CLI + GUI | CLI 与 GUI 解耦良好    |
| [`bnomad`](https://github.com/robbert229/bnomad)             | Wails GUI | Wails 调用 Go 后端逻辑 |



------

## ✅ 总结

| 类别         | 建议                                              |
| ------------ | ------------------------------------------------- |
| 结构层次     | 遵循 Clean Architecture                           |
| 模块划分     | 核心逻辑放 `core/`，协调逻辑放 `app/`             |
| 输入输出     | CLI 与 GUI 分别独立，公用 app 层接口              |
| 并发模型     | GUI 需支持异步任务，建议使用 goroutine/channel    |
| 重构难度控制 | 提前解耦 UI 与业务逻辑，有助于 CLI → GUI 平滑过渡 |