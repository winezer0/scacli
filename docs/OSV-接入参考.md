# OSV 工具接入参考（中文版）

本文面向希望在自有系统、CI/CD 或 Go 服务中接入 OSV 的开发者，介绍两条路线：调用 OSV-Scanner 二进制，或在 Go 程序中引用 OSV-SCALIBR 库并自行组织扫描流程。也说明 OSV 数据的远程查询和离线本地使用方式。

> 术语：OSV 是漏洞数据格式和数据库生态；OSV-Scanner 是现成的 CLI 扫描器；OSV-SCALIBR 是可嵌入 Go 程序的 SCA 库。OSV-SCALIBR 与 OSV-Scanner 的功能并非完全一一对应，接入前应对照所固定版本的文档。

## 1. 方案对比

| 路线 | 项目地址 | 适合场景 | 主要结果 |
|---|---|---|---|
| OSV-Scanner 二进制 | [google/osv-scanner](https://github.com/google/osv-scanner) | 多语言项目、CI/CD、希望复用成熟命令行流程 | JSON、SARIF、HTML、Markdown、文本表格等；具体以版本为准 |
| OSV-SCALIBR Go 库 | [google/osv-scalibr](https://github.com/google/osv-scalibr) | Go 服务深度集成、自定义扫描流程、插件扩展 | Go 内存中的 `*scalibr.ScanResult`；由调用方转换为 JSON、Proto 等格式 |
| OSV.dev REST API | [API 文档](https://google.github.io/osv.dev/api/) | 已有依赖清单，只需查询包版本漏洞 | OSV 漏洞记录 JSON |

只需对仓库扫描时，建议先用 OSV-Scanner。Go 库路线适合希望在同一进程内控制依赖提取和结果处理的场景；如果只是已有 SBOM/锁文件并查询漏洞，也可直接调用 OSV API。

## 2. OSV-Scanner 二进制接入

### 2.1 版本与项目地址

- 项目：[https://github.com/google/osv-scanner](https://github.com/google/osv-scanner)
- 发行版：[GitHub Releases](https://github.com/google/osv-scanner/releases)
- 安装与用法：[官方文档](https://google.github.io/osv-scanner/)
- 安装后用 `osv-scanner --version` 记录版本。CI 镜像或下载脚本应固定版本，不建议生产环境使用浮动的 `latest`。

官方 v2 安装方式示例：

```powershell
winget install Google.OSVScanner
osv-scanner --version
```

```bash
brew install osv-scanner
osv-scanner --version
```

### 2.2 结果格式

使用 `--format` 选择输出格式；机器集成优先选 JSON：

```bash
osv-scanner scan source -r . --format json --output-file osv-results.json
osv-scanner scan source -r . --format sarif --output-file osv-results.sarif
osv-scanner scan source -r . --format html --output-file osv-results.html
osv-scanner scan source -r . --format markdown --output-file osv-results.md
```

JSON 的主要层级是 `results[] → packages[] → vulnerabilities[]`。包项包括名称、版本和 ecosystem；漏洞项包括 ID、别名及 OSV 漏洞记录字段。SARIF 适合接入代码扫描平台，HTML 适合人工查看。SPDX 输出主要表达包清单，不应当作漏洞详情格式。格式和 schema 以[官方输出说明](https://google.github.io/osv-scanner/output/)为准。

### 2.3 Python 调用示例

```python
import json
import subprocess
from pathlib import Path

def scan(project_dir: str, scanner: str = "osv-scanner") -> dict:
    result_file = Path(project_dir) / "osv-results.json"
    proc = subprocess.run(
        [scanner, "scan", "source", "-r", project_dir,
         "--format", "json", "--output-file", str(result_file)],
        capture_output=True,
        text=True,
        timeout=600,
        check=False,
    )
    # OSV-Scanner 可在发现漏洞时返回退出码 1；应区分漏洞策略与执行故障。
    if proc.returncode not in (0, 1):
        raise RuntimeError(f"scanner failed ({proc.returncode}): {proc.stderr}")
    if not result_file.is_file():
        raise RuntimeError(f"scanner did not produce {result_file}: {proc.stderr}")
    return json.loads(result_file.read_text(encoding="utf-8"))
```

上线时按固定版本确认退出码语义，分别处理“扫描成功且无漏洞”“扫描成功但发现漏洞”“扫描执行失败”和超时。不要只用 `returncode == 0` 判断扫描是否成功。

### 2.4 scacli 对应代码

scacli 采用子进程调用方式，可作为外部项目接入 CLI 的参考：

- [`internal/sca/osvscanner/osvscanner.go`](../internal/sca/osvscanner/osvscanner.go)：构造参数并运行 `osv-scanner scan source -r . --format <格式> --output-file <文件>`；工作目录指向被扫描项目。
- 该适配器先读取 JSON，将 `results/packages/vulnerabilities` 映射到 scacli 的通用组件和漏洞结构；同时保留 OSV 原始 JSON。
- 根据请求再生成 HTML 或 Markdown 原生报告。OSV-Scanner 的漏洞退出码 `1` 在 scacli 中作为扫描结果处理，其余执行错误和超时作为失败。
- [`internal/sca/types.go`](../internal/sca/types.go)：统一扫描请求、结果和 `sca.Engine` 接口。
- [`internal/application/run.go`](../internal/application/run.go) 与 [`internal/application/executable.go`](../internal/application/executable.go)：选择引擎、定位二进制并装配运行流程。

简化的 Go 子进程调用形态：

```go
cmd := exec.Command(scannerPath,
    "scan", "source", "-r", ".",
    "--format", "json",
    "--output-file", "osv-results.json",
)
cmd.Dir = projectDir
cmd.Stdout = &stdout
cmd.Stderr = &stderr
err := runWithTimeout(cmd, 10*time.Minute)
```

真实实现还需按所固定版本处理漏洞退出码、超时取消、输出文件读取和 JSON 反序列化。参数使用数组传递，避免 shell 字符串拼接。

## 3. OSV-SCALIBR Go 依赖库接入

### 3.1 版本与项目地址

- 项目：[https://github.com/google/osv-scalibr](https://github.com/google/osv-scalibr)
- Go 模块：`github.com/google/osv-scalibr`
- 用项目 Release/Tag 选择固定版本，再写入 `go.mod`：

```bash
go get github.com/google/osv-scalibr@vX.Y.Z
go list -m github.com/google/osv-scalibr
```

将 `vX.Y.Z` 替换为项目实际发布的版本。SCALIBR API、插件注册名称和结果类型应按该版本核对；不要直接依赖未固定的 `main`。

### 3.2 结果格式

SCALIBR 库调用返回 `*scalibr.ScanResult`，包含扫描状态、插件状态和 inventory（包、漏洞发现等），不是 OSV-Scanner CLI 的 JSON 文本。调用方可以转换成自己的 DTO 并编码为 JSON，也可以使用项目定义的 Proto 类型；SCALIBR 自带的 `scalibr` 包装二进制示例使用 textproto。若要求与 CLI 输出兼容，应由接入方编写转换层，不能假设两者结果 schema 相同。

### 3.3 Go 引用与扫描示例

下面按官方库示例展示基本结构。插件名和插件配置应依据目标项目语言、支持清单和所固定的 SCALIBR 版本选择：

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"

    scalibr "github.com/google/osv-scalibr"
    scalibrfs "github.com/google/osv-scalibr/fs"
    pluginlist "github.com/google/osv-scalibr/plugin/list"
)

func main() {
    root := "./checkout"

    // 示例展示插件装配入口；请按目标版本的插件列表选择依赖提取插件。
    plugins, err := pluginlist.FromNames([]string{"os", "cis", "vex"})
    if err != nil {
        log.Fatal(err)
    }

    cfg := &scalibr.ScanConfig{
        ScanRoots: scalibrfs.RealFSScanRoots(root),
        Plugins:   plugins,
    }
    result := scalibr.New().Scan(context.Background(), cfg)

    // 生产代码应检查 result.Status、插件状态和部分扫描错误。
    if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
        log.Fatal(err)
    }
}
```

该代码说明 SCALIBR 的“组装插件 → 指定扫描根目录 → 执行扫描 → 处理结构化结果”方式。示例中的 `os/cis/vex` 来自官方示例，不代表它们覆盖所有语言依赖；接入项目依赖清单时，要选择匹配 manifest/lockfile 的 filesystem extractor，并确认所需漏洞 detector 已启用。插件可用项见[官方插件目录](https://github.com/google/osv-scalibr/tree/main/extractor/filesystem/list)及[库 README](https://github.com/google/osv-scalibr#how-to-use)。

### 3.4 自行查询 OSV API 的 Go 示例

若库提取出依赖清单后由业务代码查询远程 OSV 数据，可调用 `POST /v1/querybatch`。每个包需有正确的 ecosystem 和版本；名称大小写应符合 OSV ecosystem 约定。以下例子使用静态包清单演示查询与结果反序列化，实际项目可把它替换为扫描结果中的包：

```go
package osvclient

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type PackageQuery struct {
    Package struct {
        Name      string `json:"name"`
        Ecosystem string `json:"ecosystem"`
    } `json:"package"`
    Version string `json:"version"`
}

type BatchRequest struct {
    Queries []PackageQuery `json:"queries"`
}

type BatchResponse struct {
    Results []struct {
        Vulns []struct {
            ID       string    `json:"id"`
            Modified time.Time `json:"modified"`
        } `json:"vulns"`
    } `json:"results"`
}

func QueryBatch(ctx context.Context, input BatchRequest) (BatchResponse, error) {
    body, err := json.Marshal(input)
    if err != nil {
        return BatchResponse{}, fmt.Errorf("encode OSV query: %w", err)
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodPost,
        "https://api.osv.dev/v1/querybatch", bytes.NewReader(body))
    if err != nil {
        return BatchResponse{}, fmt.Errorf("create OSV request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 20 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return BatchResponse{}, fmt.Errorf("query OSV API: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return BatchResponse{}, fmt.Errorf("OSV API returned %s", resp.Status)
    }

    var result BatchResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return BatchResponse{}, fmt.Errorf("decode OSV response: %w", err)
    }
    if len(result.Results) != len(input.Queries) {
        return BatchResponse{}, fmt.Errorf("OSV response count does not match request")
    }
    return result, nil
}
```

`querybatch` 响应通常只返回漏洞 ID 和更新时间。要取得完整漏洞记录，再按 ID 请求 `GET /v1/vulns/{id}`；处理分页时也要继续提交各查询项的 `page_token`。详见 [OSV Query API](https://google.github.io/osv.dev/post-v1-query/) 和 [QueryBatch API](https://google.github.io/osv.dev/post-v1-querybatch/)。

## 4. 漏洞数据库：远程查询与本地部署

### 4.1 远程 OSV.dev API

远程查询适合常规在线环境。OSV-Scanner 默认通过 OSV.dev 查询漏洞数据库；直接使用 API 时，常用端点为：

- 单包/单版本：`POST https://api.osv.dev/v1/query`
- 多包批量查询：`POST https://api.osv.dev/v1/querybatch`
- 根据漏洞 ID 获取完整记录：`GET https://api.osv.dev/v1/vulns/{id}`

单包请求示例：

```bash
curl -sS -X POST https://api.osv.dev/v1/query \
  -H 'Content-Type: application/json' \
  -d '{"package":{"name":"jinja2","ecosystem":"PyPI"},"version":"2.4.1"}'
```

API 直接查询适用于调用方已经能准确提供 package + version 或 PURL 的场景。若需要解析目录、清单文件、传递依赖或生成多格式报告，使用 OSV-Scanner/SCALIBR 会更省开发工作。OSV API 的请求示例见[官方 Quickstart](https://google.github.io/osv.dev/quickstart/)。

### 4.2 OSV-Scanner 内置离线数据库

OSV-Scanner 官方支持离线漏洞数据库。建议采用“联网更新节点 → 内网分发 → 离线扫描节点”的部署方式：

1. 在可联网的受控任务中，用扫描器下载/更新本地数据库。
2. 将数据库缓存目录发布到内网制品库或只读共享存储。
3. 在扫描节点通过 `OSV_SCANNER_LOCAL_DB_CACHE_DIRECTORY` 指定该缓存根目录。
4. 扫描时使用 `--offline`；该模式不下载/更新数据库，也不向外发送项目或依赖信息。

```bash
# 在允许联网的维护任务中预下载/更新离线库
osv-scanner --offline-vulnerabilities --download-offline-databases ./project

# 离线扫描任务指定已分发的缓存目录
export OSV_SCANNER_LOCAL_DB_CACHE_DIRECTORY=/opt/osv-cache
osv-scanner --offline scan source -r ./project --format json --output-file result.json
```

#### 完全断网部署步骤

将扫描环境视为不允许出网的独立环境，离线扫描依赖三个预先准备好的输入：与目标操作系统/架构匹配的 `osv-scanner` 二进制、项目源码或镜像归档、项目涉及的 ecosystem 漏洞数据库快照。联网维护机只负责下载并校验这些文件，再通过制品库或受控介质传入隔离网络。数据库可用上述下载选项更新，也可手工下载；例如在 Windows 联网维护机下载 Go 数据库：

```powershell
$cache = 'D:\osv-cache'
$dbDir = Join-Path $cache 'osv-scanner\Go'
New-Item -ItemType Directory -Force $dbDir | Out-Null
Invoke-WebRequest `
  'https://osv-vulnerabilities.storage.googleapis.com/Go/all.zip' `
  -OutFile (Join-Path $dbDir 'all.zip')
```

对项目使用的每种 ecosystem 重复下载对应的 `<ECOSYSTEM>/all.zip`，保持 ZIP 原样并保留 `osv-scanner/<ecosystem>/all.zip` 目录结构。隔离扫描机设置同一个缓存根目录并运行：

```powershell
$env:OSV_SCANNER_LOCAL_DB_CACHE_DIRECTORY = 'D:\osv-cache'
.\osv-scanner.exe --offline scan source -r .\project `
  --format json --output-file .\result.json
```

该 `--offline` 模式只读本地库，不更新数据库，也不发送项目或依赖信息；缺少对应 ecosystem 数据库时会报错，不能自动回退到远程查询。将扫描节点配置为拒绝外网出站可作为断网验收：确认扫描成功、报告落盘，且扫描期间无网络放行规则。离线扫描仍需要完整的锁文件或其他可识别依赖输入；目标是镜像时，先将镜像导出为 Docker archive 并通过 `scan image --archive <tar>` 扫描。

缓存布局为 `<缓存根目录>/osv-scanner/<ecosystem>/all.zip`，例如 `.../osv-scanner/PyPI/all.zip`。也可从 OSV 官方数据桶下载各 ecosystem 的 `all.zip`，再按该目录结构放置。OSV-Scanner 离线模式与缓存位置以[官方 Offline Mode 文档](https://google.github.io/osv-scanner/usage/offline-mode/)为准。离线数据库是数据快照，运维方需要制定更新频率、镜像分发、校验和回滚策略；离线扫描不会自动获得最新漏洞数据。

#### 需要额外联网的功能

`--offline-vulnerabilities` 只让**漏洞数据库匹配**使用本地数据，并不等于整个扫描过程断网。官方文档明确列出的联网场景如下：

| 功能 | 需要访问的服务 | 离线时的处理 |
|---|---|---|
| 默认漏洞匹配 | OSV.dev API | 使用 `--offline` 读取本地数据库 |
| Go call analysis（Go 调用链分析） | 默认访问 Go Vulnerability Database `vuln.go.dev`；官方说明只发送已知存在漏洞的模块路径，不发送源码 | 完整 `--offline` 禁止联网；也可使用 `--no-call-analysis=go` 关闭 Go 调用链分析 |
| Maven `pom.xml` 传递依赖解析 | 默认使用 deps.dev API；可切换到 Maven Central 或私有 Maven registry | 完整 `--offline` 会禁用该解析；只要已有解析后的 lockfile，通常可直接扫描 lockfile；也可用 `--no-resolve` 关闭解析 |
| C/C++ vendored 依赖版本识别 | OSV determineversion API | 官方离线模式不支持 commit-level 扫描；离线任务应提供明确 package/version 的依赖清单，或排除此类识别 |
| `scan image <name:tag>` 镜像获取 | 镜像不在本机时，Docker 会从对应 registry 拉取 | 先导出镜像 tar，再使用 `scan image --archive <文件>`，可避免扫描时访问镜像仓库 |
| `fix` 引导式修复 | 默认使用 deps.dev 解析依赖图；重锁/升级还会访问包管理器 registry | 离线漏洞库只覆盖漏洞查询，不会提供依赖图或包仓库；离线环境不要把 `fix` 当作纯离线操作 |
| OSV-SCALIBR/OSV-Scanner 插件 | 仅要求在线能力的插件可能访问外部服务 | 库接入时按插件文档检查 `Network` capability，并通过 `plugin.FilterByCapabilities` 过滤；CLI 会按能力自动筛选插件 |

具体依据：官方说明 [Go call analysis 使用 govulncheck](https://google.github.io/osv-scanner/usage/)、[govulncheck 默认请求 vuln.go.dev 且不上传源码](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)、[Maven 传递依赖扫描默认查询 deps.dev](https://google.github.io/osv-scanner/supported-languages-and-lockfiles/#transitive-dependency-scanning)、[vendored C/C++ 扫描调用 determineversion API，离线模式不支持 commit-level 扫描](https://google.github.io/osv-scanner/supported-languages-and-lockfiles/#vendored-dependencies)、[容器扫描的本地镜像与 tar archive 方式](https://google.github.io/osv-scanner/usage/scan-image)，以及 [`fix` 的 deps.dev 和 registry 数据源](https://google.github.io/osv-scanner/experimental/guided-remediation/#data-source)。SCALIBR 支持按插件能力筛选，见[官方库示例](https://github.com/google/osv-scalibr#running-built-in-plugins)。

需要严格断网时，使用 `--offline`、预先准备扫描输入和数据库，并避免需要传递依赖解析、C/C++ 版本识别、远程镜像拉取或在线修复的路径。若只使用 `--offline-vulnerabilities`，仍应为依赖解析、镜像仓库和已启用插件配置网络访问策略。

### 4.3 自托管远程 API/数据库

如果需求是“内网服务统一提供查询 API”，需要区分“离线文件库”和“自托管 OSV.dev 服务”：

- 最简单、由 OSV-Scanner 官方文档直接支持的方式是镜像离线数据库并使用 `--offline`。
- 如果自行运行兼容 OSV API 的服务，调用方可以把上面的 REST 示例中的 API base URL 改为内网地址；API 兼容性、数据同步和可用性由服务运维方负责。
- OSV.dev 源码仓库的官方架构部署到 Google Cloud，包含 API、数据库、索引和数据导出等多个服务，不是一个开箱即用的本地数据库容器。自托管完整服务应评估其 [OSV.dev 架构](https://google.github.io/osv.dev/architecture/)和[服务端代码](https://github.com/google/osv.dev)。
- 本参考不假设 OSV-Scanner CLI 可通过参数直接切换任意私有 API 地址；如需自定义 API endpoint，应使用自有 API 客户端，或先验证固定版本的 scanner 是否提供对应配置。

如果安全要求是数据不出内网，优先采用官方离线库快照；若要求所有调用方统一访问内网 HTTP API，则需部署并维护兼容 API 的服务，并确保同步 OSV 格式漏洞记录。

## 5. 接入验收要点

- 固定并记录 OSV-Scanner 或 OSV-SCALIBR 版本；锁定 Go 模块依赖。
- 准确映射包名、版本、ecosystem/PURL。ecosystem 拼写与大小写错误会导致漏报。
- 分开处理无漏洞、发现漏洞、扫描不完整/失败、网络错误和超时。
- 输出原始结果和归一化结果时保留来源文件、包版本、漏洞 ID、别名和修复版本等字段。
- 对离线库记录获取时间、数据版本/摘要和部署范围，设置自动更新与过期提醒。
- 报告可能包含内部包名和目录路径，按内部数据分级要求保存。
