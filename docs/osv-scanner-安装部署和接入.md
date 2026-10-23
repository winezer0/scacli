# OSV-Scanner 安装部署和接入

## 工具简介

OSV-Scanner 是 Google 开源的依赖漏洞扫描器。它从源码目录、锁文件或 SBOM 提取依赖，再与 OSV 漏洞数据库匹配。命令行工具不需要 API Token，适合开发机和 CI/CD 使用。它以 CLI 为主要集成方式；其他程序通常通过子进程调用命令并解析 JSON。

## 安装

优先从[官方安装说明](https://google.github.io/osv-scanner/installation/)选择平台对应的包管理器或二进制发行包。安装后验证：

```powershell
osv-scanner --version
```

常见安装方式：

```powershell
# Windows
winget install Google.OSVScanner
# 或 Scoop
scoop install osv-scanner
```

```bash
# macOS
brew install osv-scanner

# Go 环境安装（需要官方说明中要求的 Go 版本）
go install github.com/google/osv-scanner/v2/cmd/osv-scanner@latest
```

Linux 用户可下载[官方 Releases](https://github.com/google/osv-scanner/releases)中的对应二进制，或使用发行版仓库支持的包。部署到 CI 时，建议固定版本并校验下载来源；不要依赖 `latest` 以免构建结果随时间变化。

## 本地扫描

扫描整个源码目录并写入 JSON：

```bash
osv-scanner scan source -r . --format json --output-file osv-results.json
```

HTML 和 Markdown 输出可按需指定：

```bash
osv-scanner scan source -r . --format html --output-file osv-results.html
osv-scanner scan source -r . --format markdown --output-file osv-results.md
```

OSV-Scanner v2 的子命令、输出参数和格式以[官方 Usage](https://google.github.io/osv-scanner/usage/)及[输出格式说明](https://google.github.io/osv-scanner/output/)为准。源码扫描支持哪些清单与锁文件，参见[官方支持列表](https://google.github.io/osv-scanner/supported-languages-and-lockfiles/)。

## 代码接入

建议先把待扫描仓库检出到隔离工作目录，再用参数数组调用可执行文件。不要拼接 shell 命令字符串，避免路径空格或特殊字符导致调用错误，也避免命令注入。

### Python 示例

```python
import json
import subprocess
from pathlib import Path

def scan(project_dir: str, scanner: str = "osv-scanner") -> dict:
    output = Path(project_dir) / "osv-results.json"
    proc = subprocess.run(
        [scanner, "scan", "source", "-r", project_dir,
         "--format", "json", "--output-file", str(output)],
        text=True,
        capture_output=True,
        check=False,
        timeout=600,
    )
    # 扫描到漏洞时 CLI 可能返回非零状态；先检查结果文件是否生成，
    # 再结合所用版本文档判断退出码是“发现漏洞”还是“执行失败”。
    if not output.is_file():
        raise RuntimeError(f"OSV-Scanner failed ({proc.returncode}): {proc.stderr}")
    return json.loads(output.read_text(encoding="utf-8"))
```

生产接入还应检查 JSON 解析错误、超时和诊断输出，并记录扫描器版本。若以退出码阻断流水线，应按固定版本的 CLI 行为区分“发现漏洞”和“工具运行失败”。

### Go 调用方式

scacli 使用 `os/exec` 的参数列表调用 CLI，并设定工作目录和超时；不是把 OSV-Scanner 作为 Go 库链接进程内：

```go
cmd := exec.Command(scannerPath,
    "scan", "source", "-r", ".",
    "--format", "json",
    "--output-file", "osv-output.json",
)
cmd.Dir = projectDir
err := runWithTimeout(cmd, 10*time.Minute)
```

这里的 `runWithTimeout` 应负责超时取消、等待子进程和回收资源。读取报告后用 `encoding/json` 解码，并根据业务需要保留原始 JSON。scacli 的具体适配器见 [`internal/sca/osvscanner/osvscanner.go`](../internal/sca/osvscanner/osvscanner.go)。

## scacli 如何接入

scacli 的适配器实现位于 [`internal/sca/osvscanner/osvscanner.go`](../internal/sca/osvscanner/osvscanner.go)：

1. 应用层解析配置，按 `PATH`、程序目录、程序目录下的 `bin` 查找 `osv-scanner`，也允许显式指定路径。
2. 每个 Git 项目先检出到工作目录，再通过 `exec.Command` 在该目录执行 `osv-scanner scan source -r .`。
3. 先生成并读取 `osv-output.json`，将 OSV 结果归一化到 scacli 的组件/漏洞 DTO，同时保留原始 JSON。
4. 根据请求格式额外生成 HTML 或 Markdown 原生报告，再由报告层写入统一报告目录。
5. OSV-Scanner 返回“发现漏洞”的退出状态不能简单视为进程崩溃；适配器对退出码 `1` 特殊处理，其余异常、超时或缺少报告仍记为失败。

引擎通过 [`internal/sca/types.go`](../internal/sca/types.go) 中的 `sca.Engine` 接口接入。要新增或替换扫描器时，沿用“配置/路径解析 → 实现 `Scan(Request)` → 返回标准结果和原始报告 → 报告层落盘”的边界。

## CI/CD 示例

```yaml
# GitHub Actions 可复用工作流示例；请在仓库实际 workflow 文件中使用。
jobs:
  scan:
    permissions:
      actions: read
      contents: read
      security-events: write
    uses: google/osv-scanner-action/.github/workflows/osv-scanner-reusable.yml@v2.7.0
    with:
      scan-args: |-
        --recursive
        ./
```

官方也维护 PR 扫描、定期扫描等 GitHub Action 工作流。版本、权限和参数请查看[官方 Action 文档](https://google.github.io/osv-scanner/github-action/)。

## 部署与安全注意事项

- 将扫描器版本固定在安装脚本、容器镜像或工具缓存中，并在日志中记录版本。
- CI 作业需要网络访问漏洞数据源；若环境离线，按[官方离线漏洞库文档](https://google.github.io/osv-scanner/usage/)准备本地数据库。
- 把源码工作目录设为只读挂载（若扫描器无需写入），报告写到独立目录，并避免并行作业共用输出文件名。
- 报告可能包含内部依赖名称、版本和路径，应按内部安全策略保存与传递。
