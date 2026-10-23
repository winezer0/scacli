# OpenSCA 安装部署和接入

## 工具简介

OpenSCA-cli 是开源的软件成分分析工具，可识别依赖、漏洞和许可证信息，并生成报告/SBOM。它可本地运行，也可配置云端漏洞库；访问云端服务时需要相应 Token。命令参数、配置字段和输出格式以[官方仓库](https://github.com/XmirrorSecurity/OpenSCA-cli)对应版本文档为准。

## 安装部署

可从[官方 Releases](https://github.com/XmirrorSecurity/OpenSCA-cli/releases)下载对应系统和架构的发行包并解压，也可使用安装脚本或包管理器。常见安装方式：

```bash
# macOS
brew install opensca-cli

# Linux/macOS 安装脚本
curl -sSL https://raw.githubusercontent.com/XmirrorSecurity/OpenSCA-cli/master/scripts/install.sh | sh
```

```powershell
# Windows PowerShell
iex "&{$(irm https://raw.githubusercontent.com/XmirrorSecurity/OpenSCA-cli/master/scripts/install.ps1)}"

# 或 Winget
winget install OpenSCA-cli
```

也可使用 Docker：

```bash
docker run --rm -v "$PWD:/src" opensca/opensca-cli -path /src -out /src/output.html
```

部署后检查 `opensca-cli --help`，并按目标版本文档确认参数名。OpenSCA-cli 的安装和 Docker 示例参见[官方 README](https://github.com/XmirrorSecurity/OpenSCA-cli#installation)。

## 配置与扫描

最小扫描命令：

```bash
opensca-cli -path ./project -token "$OPENSCA_TOKEN" -out ./reports/result.html
```

`-path` 指目标目录，`-token` 用于云服务验证，`-out` 根据文件后缀生成报告。版本支持的格式可能包括 HTML、JSON、XML、SARIF 以及 SBOM 格式；使用前确认目标版本支持该后缀。OpenSCA 可用 `-config` 指定配置文件；复杂扫描参数（如忽略路径、漏洞库地址等）放在官方配置文件中，不要假设不同主版本字段完全相同。

## 代码接入

OpenSCA-cli 通常通过命令行接入。调用方负责准备源码目录、创建输出目录、运行进程、设置超时并检查报告文件。使用参数数组而不是 shell 字符串，Token 从 CI Secret 或密钥服务注入。

### Python 示例

```python
import os
import subprocess
from pathlib import Path

def scan(project_dir: str, report_dir: str, scanner: str = "opensca-cli") -> Path:
    report = Path(report_dir) / "opensca-report.html"
    report.parent.mkdir(parents=True, exist_ok=True)
    token = os.environ["OPENSCA_TOKEN"]
    proc = subprocess.run(
        [scanner, "-token", token, "-path", project_dir, "-out", str(report)],
        text=True,
        capture_output=True,
        check=False,
        timeout=600,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"OpenSCA failed ({proc.returncode}): {proc.stderr}")
    if not report.is_file():
        raise RuntimeError("OpenSCA completed without producing the expected report")
    return report
```

### Go 调用方式

scacli 会先删除工作目录内旧的 `output.html`，然后在扫描目录中执行 OpenSCA，最后读取 HTML 原始报告：

```go
cmd := exec.Command(scannerPath,
    "-token", token,
    "-path", projectDir,
    "-out", "output.html",
)
cmd.Dir = projectDir
err := runWithTimeout(cmd, 10*time.Minute)
if err != nil {
    return err
}
report, err := os.ReadFile(filepath.Join(projectDir, "output.html"))
```

调用时要检查进程错误、超时和输出文件，不能仅以退出码为成功标准。若扫描报告格式改为 JSON/SARIF/SBOM，应相应更新结果解析和归一化逻辑。适配器代码见 [`internal/sca/opensca/opensca.go`](../internal/sca/opensca/opensca.go)。

## scacli 如何接入

scacli 的 OpenSCA 接入采用 CLI 适配器，而不是直接导入 OpenSCA 的 Go 包：

1. `scan.engine` 选择 `opensca` 后，应用层定位 `opensca-cli` 可执行程序。
2. 每个 Git 仓库进入独立工作目录，调用 `opensca-cli -token TOKEN -path PROJECT_PATH -out output.html`，超时时间由 `scan.scan_timeout` 控制。
3. 适配器先清理同名旧报告，进程成功后读取新生成的 HTML，保存在 `NativeReports.HTML`。
4. 报告层把原生 HTML 报告复制到统一报告目录。当前 scacli 的 OpenSCA 适配器只消费 HTML，不解析其组件/漏洞为统一结果；不要把 OpenSCA 的其他格式误认为 scacli 当前已接入。

相关代码：[`internal/application/run.go`](../internal/application/run.go)、[`internal/application/executable.go`](../internal/application/executable.go)、[`internal/sca/opensca/opensca.go`](../internal/sca/opensca/opensca.go) 和 [`internal/sca/types.go`](../internal/sca/types.go)。统一配置支持 `OPENSCA_TOKEN`，并可通过 `scan.opensca` 或 `--opensca` 指定程序路径。配置中不建议明文保存 Token。

## CI/CD 示例

使用 Linux runner 的通用示例，Token 应配置为流水线 Secret：

```yaml
steps:
  - name: Install OpenSCA-cli
    run: curl -sSL https://raw.githubusercontent.com/XmirrorSecurity/OpenSCA-cli/master/scripts/install.sh | sh
  - name: Scan dependencies
    env:
      OPENSCA_TOKEN: ${{ secrets.OPENSCA_TOKEN }}
    run: |
      mkdir -p reports
      opensca-cli -path "$PWD" -token "$OPENSCA_TOKEN" -out "$PWD/reports/opensca.html"
```

如使用 Docker，须把待扫描目录挂载到容器并将容器内路径传给 `-path`；报告输出到挂载目录，流水线才能收集文件。更多集成示例见[官方 CI/CD 指南](https://github.com/XmirrorSecurity/OpenSCA-cli/blob/master/docs/Integrations/CICD.md)。

## 部署注意事项

- 使用固定版本的可执行文件或容器镜像，记录版本并安排升级。
- 按实际部署选择云端或本地漏洞库，并确保网络、代理、证书和凭据配置满足环境要求。
- 扫描路径、报告输出路径和配置文件路径应明确指定；并行扫描使用独立工作目录，避免共享 `output.html`。
- 报告和 Token 按敏感数据管理；不要将 Token 写入镜像层、仓库或 CI 日志。
