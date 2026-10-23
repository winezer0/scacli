# MurphySec 安装部署和接入

## 工具简介

MurphySec CLI 对项目依赖进行识别，并连接 MurphySec 服务查询风险。官方说明 CLI 会上传依赖信息和项目基本信息用于匹配，不上传代码片段；使用前应确认组织允许将依赖元数据发送到服务端。CLI 需要 MurphySec 账号的 Access Token。详见[官方仓库说明](https://github.com/murphysecurity/murphysec)。

## 安装部署

按[官方安装指南](https://github.com/murphysecurity/murphysec#1-install-murphysec-cli)安装；安装脚本会从网络获取程序，生产部署可改用官方 Releases 中经审核的固定版本。

```bash
# Linux
wget -q https://s.murphysec.com/release/install.sh -O - | /bin/bash

# macOS
curl -fsSL https://s.murphysec.com/release/install.sh | /bin/bash
```

```powershell
# Windows PowerShell
powershell -Command "iwr -useb https://s.murphysec.com/release/install.ps1 | iex"
```

确认命令可用并查看版本：

```bash
murphysec --version
```

从 MurphySec 平台获取 Access Token。交互式机器可用 `murphysec auth login` 保存认证；CI 环境建议通过密钥管理服务注入 Token，并在命令参数中传入，不要提交到代码库或打印到日志。

## 本地扫描

```bash
murphysec scan ./project --token "$MURPHYSEC_TOKEN" --json
```

PowerShell：

```powershell
$env:MURPHYSEC_TOKEN = "<从密钥管理器注入>"
murphysec scan . --token $env:MURPHYSEC_TOKEN --json
```

`--json` 用于机器读取结果；未要求 JSON 时，CLI 默认不展示详细结果，结果可在 MurphySec 平台查看。参数和服务端行为可能随 CLI 版本调整，接入前查看[官方 CLI 文档](https://github.com/murphysecurity/murphysec)。

## 代码接入

不同构建系统都可以采用相同流程：检出源码 → 注入 Token → 调用 CLI → 解析 JSON → 保存报告或按策略阻断流水线。使用参数数组，不通过 shell 字符串拼命令。

### Python 示例

```python
import json
import os
import subprocess

def scan(project_dir: str, scanner: str = "murphysec") -> dict:
    token = os.environ["MURPHYSEC_TOKEN"]
    proc = subprocess.run(
        [scanner, "--token", token, "scan", project_dir, "--json"],
        text=True,
        capture_output=True,
        check=False,
        timeout=600,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"MurphySec failed ({proc.returncode}): {proc.stderr}")
    try:
        return json.loads(proc.stdout)
    except json.JSONDecodeError as exc:
        raise RuntimeError("MurphySec returned invalid JSON") from exc
```

对某一固定 CLI 版本做一次成功与失败场景验证，明确其退出码语义；日志中只记录退出码、版本和脱敏后的错误，不记录完整命令参数（其中含 Token）。

### Go 调用方式

scacli 用参数数组运行 `murphysec --token TOKEN scan <项目目录名> --json`，并用 JSON 解码兼容根对象或 `data` 包装结构：

```go
cmd := exec.Command(scannerPath,
    "--token", token,
    "scan", filepath.Base(projectDir), "--json",
)
cmd.Dir = filepath.Dir(projectDir)
cmd.Env = append(os.Environ(), "SCACLI_TOKEN=")
err := runWithTimeout(cmd, 10*time.Minute)
```

随后读取标准输出并解析 JSON。不要把 Token 拼进单条 shell 命令；也不要将完整参数、环境变量或含密钥的错误内容写入日志。scacli 的实现见 [`internal/sca/murphysec/murphysec.go`](../internal/sca/murphysec/murphysec.go)。

## scacli 如何接入

scacli 将 MurphySec CLI 包装为通用扫描器接口，接入位置如下：

- [`internal/sca/types.go`](../internal/sca/types.go) 定义请求、统一组件/漏洞结果、原始报告和 `sca.Engine` 接口。
- [`internal/sca/murphysec/murphysec.go`](../internal/sca/murphysec/murphysec.go) 实现 `Scan(Request)`：在项目父目录设定工作目录，以项目目录名调用 MurphySec，限制扫描时间，读取 JSON 并保留原始 JSON。
- [`internal/application/run.go`](../internal/application/run.go) 根据 `scan.engine: murphysec` 创建适配器；[`internal/application/executable.go`](../internal/application/executable.go) 解析二进制位置。
- [`config/config.go`](../config/config.go) 支持 `MURPHYSEC_TOKEN`、通用 `SCACLI_TOKEN` 和配置文件。项目优先级为 CLI 参数 > 引擎专用环境变量 > 通用环境变量 > 配置文件。
- 统一报告层把标准结果写入 scacli 报告目录；MurphySec 原始 JSON 同时保留。

配置示例：

```yaml
scan:
  engine: murphysec
  token: "" # 建议使用 MURPHYSEC_TOKEN 注入
  murphysec: "" # 留空则按 PATH、程序目录、程序目录/bin 查找
  scan_timeout: 600
```

## CI/CD 示例

下面为通用 shell 步骤；将 Token 设置为 CI 的受保护 Secret：

```yaml
steps:
  - name: Install MurphySec CLI
    run: curl -fsSL https://s.murphysec.com/release/install.sh | sh
  - name: Scan dependencies
    env:
      MURPHYSEC_TOKEN: ${{ secrets.MURPHYSEC_TOKEN }}
    run: murphysec --token "$MURPHYSEC_TOKEN" scan . --json > murphysec-results.json
```

官方提供 [Jenkins 集成指南](https://www.murphysec.com/docs/)，也可在其他流水线调用相同 CLI。上线时应固定 CLI 版本、限制报告访问权限，并确认依赖元数据外传符合组织要求。
