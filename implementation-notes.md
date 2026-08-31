# 实现说明

## 分包决策

本次重构将根目录的扫描、Git 和报告代码按稳定业务边界拆分，CLI 入口和组合编排位于 `cmd/scacli`：

| 包 | 职责 | 主要 API |
|---|---|---|
| `config` | 配置文件、环境变量、命令行覆盖和跨领域校验 | `Load`、`Validate` |
| `internal/application` | 组合配置、选择引擎、逐项目执行和 CLI 输入解析 | `Run`、`Options` |
| `internal/git` | 仓库身份、认证环境、浅 Clone、复用和更新运行目录 | `NewClient`、`Prepare`、`Clone`、`Update` |
| `internal/tasks` | 扫描任务列表 DTO | `Task`、`Config` |
| `internal/sca` | 引擎无关的扫描请求、标准化结果和原生报告 DTO | `Engine`、`EngineResult` |
| `internal/sca/murphysec` | MurphySec CLI 适配和 JSON 解析 | `New`、`Engine.Scan` |
| `internal/sca/opensca` | OpenSCA CLI 适配和 HTML 读取 | `New`、`Engine.Scan` |
| `internal/sca/osvscanner` | OSV-Scanner CLI 适配、JSON 解析和原生格式生成 | `New`、`Engine.Scan` |
| `internal/report` | 格式选择、报告命名、原生报告输出和失败兜底报告 | `ResolveFormats`、`WriteReports` |
| `internal/process` | 外部进程超时和诊断信息处理 | `RunWithTimeout`、`DiagnosticOutput` |

依赖方向为：`cmd/scacli → config/application → 业务包 → process`。`config` 统一加载 `config.yaml` 和 `task.yaml`，但任务 DTO 归属 `internal/tasks`。扫描引擎不依赖 `config`，报告包不依赖 CLI，避免配置包与业务包循环依赖。每个引擎实例持有已解析的可执行文件路径，扫描请求只传递工作目录、Token、超时和输出格式。
