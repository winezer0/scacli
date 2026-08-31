# scacli

用 Go 编写的软件成分分析（SCA）命令行工具

可以批量克隆 Git 仓库，并调用 MurphySec、OpenSCA 或 OSV-Scanner 扫描依赖风险、生成报告。

## 配置文件

默认读取当前目录或程序所在目录下的 `config.yaml`，也可以显式指定：

```powershell
scacli.exe --config .\config.yaml
```

示例：

```yaml
scan:
  engine: opensca
  token: ""
  opensca: "C:/Tools/opensca-cli.exe"
  osv_scanner: "C:/Tools/osv-scanner.exe"
  scan_timeout: 600

git:
  ssh_key: "C:/Users/windows/.ssh/id_rsa"
  user: ""
  password: ""
  clone_timeout: 300

report:
  output: reports
```

任务列表单独保存在 `task.yaml`：

```yaml
tasks:
  - url: git@gitlab.example.com:team/app.git
    branch: main
```

### 配置字段说明

#### `scan`

| 字段 | 类型 | 默认值 | 说明与可选值 |
|---|---|---|---|
| `scan.engine` | 字符串 | `murphysec` | 扫描引擎。可选：`murphysec`、`opensca`、`osv-scanner`。 |
| `scan.token` | 字符串 | 空 | 扫描引擎 Token。建议留空，优先使用环境变量。 |
| `scan.murphysec` | 字符串 | 空 | MurphySec 可执行文件路径。为空时按 `PATH`、程序目录、程序目录/bin 查找。 |
| `scan.opensca` | 字符串 | 空 | OpenSCA 可执行文件路径。为空时按 `PATH`、程序目录、程序目录/bin 查找。 |
| `scan.osv_scanner` | 字符串 | 空 | OSV-Scanner 可执行文件路径。为空时按 `PATH`、程序目录、程序目录/bin 查找。OSV-Scanner 不需要 Token。 |
| `scan.scan_timeout` | 整数 | `600` | 扫描超时时间，单位为秒，必须大于 0。 |

#### `git`

| 字段 | 类型 | 默认值 | 说明与可选值 |
|---|---|---|---|
| `git.ssh_key` | 字符串 | 空 | SSH 私钥文件路径。为空时继承系统 SSH Agent、SSH 配置和默认私钥。 |
| `git.user` | 字符串 | 空 | HTTPS Git 用户名；SSH Clone 不需要配置。 |
| `git.password` | 字符串 | 空 | HTTPS Git 密码或访问 Token。建议使用系统凭据管理器或环境变量，避免明文保存。 |
| `git.clone_timeout` | 整数 | `300` | Git Clone 超时时间，单位为秒，必须大于 0。 |

#### `report`

| 字段 | 类型 | 默认值 | 说明与可选值 |
|---|---|---|---|
| `report.output` | 字符串 | `reports` | JSON、HTML 和 Markdown 报告输出目录；支持相对路径和绝对路径。不能为空。 |

#### `task.yaml`

| 字段 | 类型 | 是否必填 | 说明 |
|---|---|---|---|
| `tasks` | 数组 | 否 | 任务文件中的项目列表。命令行指定 `--git` 或 `--git-file` 时，该列表不使用。 |
| `tasks[].url` | 字符串 | 是 | Git 仓库地址，支持 SSH 或 HTTPS 地址。 |
| `tasks[].branch` | 字符串 | 否 | 分支或 Tag。未配置时使用远程仓库默认分支。 |

### 配置覆盖规则

配置值的优先级从高到低为：

```text
命令行参数 > 当前引擎专用 Token 环境变量 > SCACLI_TOKEN > config.yaml > 内置默认值
```

其中：

- `--token` 覆盖引擎专用 Token 环境变量、`SCACLI_TOKEN` 和 `scan.token`。
- MurphySec 使用 `MURPHYSEC_TOKEN`，OpenSCA 使用 `OPENSCA_TOKEN`；未设置专用变量时回退到 `SCACLI_TOKEN`。
- `--engine`、`--output`、`--ssh-key` 等命令行参数覆盖对应配置文件字段。
- `--git` 或 `--git-file` 覆盖 `task.yaml` 中的 `tasks`。

任务来源优先级为：`--git` / `--git-file` > `task.yaml`。运行配置优先级为：命令行参数 > 环境变量 > `config.yaml` > 默认值。

`scacli` 是基于 Go 的 SCA 命令行工具，支持逐个 Clone Git 项目，并使用 MurphySec、OpenSCA 或 OSV-Scanner 执行扫描。

## 项目结构

根目录的 `main.go` 仅负责 CLI 解析和配置装配。核心实现按职责位于 `internal/application`、`internal/git`、`internal/sca`、`internal/report` 和 `internal/process`；各扫描引擎分别位于 `internal/sca/murphysec`、`internal/sca/opensca` 和 `internal/sca/osvscanner`。

## 编译

```powershell
go build -o scacli.exe ./cmd/scacli
```

## 基本用法

扫描单个项目：

```powershell
scacli.exe --git "git@gitlab.example.com:team/app.git" --token "<TOKEN>"
```

扫描多个项目：

```powershell
scacli.exe -g "git@gitlab.example.com:team/app1.git" -g "git@gitlab.example.com:team/app2.git" -t "<TOKEN>"
```

从文件读取：

```powershell
scacli.exe --git-file repositories.txt --output reports --token "<TOKEN>"
```

使用 `task.yaml`：

```powershell
scacli.exe --config .\config.yaml --task .\task.yaml
```

未指定 `--task` 时，程序按当前目录、程序目录的顺序查找 `task.yaml`。

地址文件每行一个项目，也可以在地址后指定分支或 Tag：

```text
git@gitlab.example.com:team/app1.git
git@gitlab.example.com:team/app2.git develop
git@gitlab.example.com:team/app3.git release/1.0
```

## Git Clone

默认使用浅克隆：

```text
git clone --quiet --depth 1 <地址> <目录>
```

全局指定分支：

```powershell
scacli.exe -g "git@gitlab.example.com:team/app.git" -b develop -t "<TOKEN>"
```

地址文件中的分支优先于 `--branch`。

## SSH 认证

```powershell
scacli.exe `
  --git "git@gitlab.example.com:team/app.git" `
  --ssh-key "C:\Users\windows\.ssh\id_rsa" `
  --token "<TOKEN>"
```

不指定 `--ssh-key` 时，继承系统 Git/SSH 配置，包括 SSH Agent、`~/.ssh/config` 和默认私钥。公钥需要提前配置到 Git 服务端。

## HTTPS 认证

```powershell
scacli.exe `
  --git "https://gitlab.example.com/team/app.git" `
  --git-user "secteam" `
  --git-password "<GIT_TOKEN>" `
  --token "<SCA_TOKEN>"
```

也可以使用环境变量 `GIT_PASSWORD` 提供 Git 密码或 Token。

## 扫描引擎

### MurphySec

```powershell
scacli.exe --engine murphysec --git <Git地址> --token <TOKEN>
```

执行：

```text
murphysec --token TOKEN scan PROJECT_PATH --json
```

### OpenSCA

```powershell
scacli.exe --engine opensca --git <Git地址> --token <TOKEN>
```

执行：

```text
opensca-cli -token TOKEN -path PROJECT_PATH -out output.html
```

工具读取运行目录中的 `output.html`，并将 OpenSCA 原始 HTML 独立输出到 `reports`。OpenSCA 当前仅提供 HTML 报告。

### OSV-Scanner

```powershell
scacli.exe --engine osv-scanner --git <Git地址>
```

工具调用 `osv-scanner scan source -r <项目目录> --format json`，解析组件、漏洞 ID、别名、摘要、详情和严重级别，并写入 JSON/Markdown 报告。OSV-Scanner 不需要配置 Token。

OSV-Scanner 还会按请求生成原生 HTML 和 Markdown 报告。

## 二进制文件查找

未指定扫描器路径时，按顺序查找对应的可执行文件：

1. 系统 `PATH`
2. `scacli` 所在目录
3. `scacli` 所在目录的 `bin` 子目录

Windows 会自动尝试 `.exe` 后缀。也可以显式指定：

```powershell
scacli.exe --murphysec "C:\Tools\murphysec.exe"
scacli.exe --opensca "C:\Tools\opensca-cli.exe"
scacli.exe --osv-scanner "C:\Tools\osv-scanner.exe"
```

## Token

建议不要在 `config.yaml` 中明文保存 Token，优先使用环境变量。优先级如下：

```text
--token > MURPHYSEC_TOKEN / OPENSCA_TOKEN > SCACLI_TOKEN > scan.token
```

根据引擎设置对应变量：

```powershell
$env:MURPHYSEC_TOKEN = "<MURPHYSEC_TOKEN>"
$env:OPENSCA_TOKEN = "<OPENSCA_TOKEN>"
scacli.exe --engine opensca --git <Git地址>
```

也可以使用通用变量：

```powershell
$env:SCACLI_TOKEN = "<TOKEN>"
scacli.exe --engine opensca --git <Git地址>
```

## 报告

默认输出到 `reports`，可使用 `--output` 修改：

`--format` 支持逗号分隔或重复传参，例如：

```powershell
scacli.exe --format json,html,md --format md
```

未指定 `--format` 时默认请求 `json,html,md`，程序会根据引擎能力过滤不支持的格式：

| 引擎 | JSON | HTML | Markdown |
|---|---:|---:|---:|
| MurphySec | 支持 | 不输出 | 不输出 |
| OpenSCA | 不输出 | 支持 | 不输出 |
| OSV-Scanner | 支持 | 支持 | 支持 |

每种输出格式都会写入完整内容，不通过引用其他格式文件代替报告内容。

```text
reports/<仓库路径>_<项目名>@<分支或Tag>-<YYYYMMDD_HHmmss>.md
reports/<仓库路径>_<项目名>@<分支或Tag>-<YYYYMMDD_HHmmss>.json
reports/<仓库路径>_<项目名>@<分支或Tag>-<YYYYMMDD_HHmmss>.html
```

JSON、HTML 和 Markdown 报告均写入各自完整的数据内容，不通过引用其他报告文件代替。OpenSCA 的 HTML 为原始审计报告；OSV-Scanner 的三种格式均由其原生输出独立生成。

## 主要参数

| 参数 | 说明 |
|---|---|
| `-c, --config` | 配置文件路径，默认自动查找 `config.yaml` |
| `-T, --task` | 扫描任务文件路径，默认自动查找 `task.yaml` |
| `--engine` | `murphysec`、`opensca` 或 `osv-scanner` |
| `-g, --git` | Git 地址，可重复 |
| `-f, --git-file` | Git 地址文件 |
| `-b, --branch` | 全局分支或 Tag |
| `-t, --token` | SCA Token |
| `-o, --output` | 报告目录，默认 `reports` |
| `-F, --format` | 输出格式：`json`、`html`、`md`；支持逗号分隔和重复传参 |
| `-m, --murphysec` | MurphySec 路径 |
| `--opensca` | OpenSCA 路径 |
| `--osv-scanner` | OSV-Scanner 路径 |
| `-k, --ssh-key` | SSH 私钥路径 |
| `--git-user` | HTTPS Git 用户名 |
| `--git-password` | HTTPS Git 密码/Token |
| `--clone-timeout` | Clone 超时，默认 300 秒 |
| `--scan-timeout` | 扫描超时，默认 600 秒 |

## Clone 工作目录

每个项目固定 Clone 到程序所在目录的 `runtimes` 子目录：

```text
程序目录/runtimes/仓库路径_项目名称@分支或Tag
```

仓库路径、项目名称和分支或 Tag 中的不可用字符会替换为 `_`。再次扫描时，程序会先检查同名目录的 `origin` 是否与当前 Git 地址一致；一致则执行 `git fetch --depth=1` 和 `git reset --hard FETCH_HEAD` 后复用。远程地址不一致时，自动追加 `-2`、`-3` 等后缀避免误覆盖。HTML、JSON、Markdown 报告使用同一命名主体并追加当前时间，格式为 `YYYYMMDD_HHmmss`。`git reset --hard` 会丢弃该 Clone 目录中的已跟踪本地修改。Clone 目录和 OpenSCA 的 `output.html` 默认都会保留，需由用户自行删除。

## 失败处理

单个项目 Clone、认证或扫描失败时会生成失败报告，并继续处理其他项目。全部项目处理完成后，只要存在失败项目，程序以非零状态码退出。

## 安全建议

- 不要将 Token、Git 密码或私钥提交到仓库
- 优先使用环境变量或 CI/CD Secret
- 避免在共享日志中输出 Token
