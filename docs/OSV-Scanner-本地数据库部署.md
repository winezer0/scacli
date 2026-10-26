# OSV-Scanner 本地数据库部署

本文介绍一种简单部署方式：使用 OSV-Scanner CLI 扫描项目，扫描时从本地漏洞数据库读取数据；单独安排联网任务定期更新数据库，避免每次扫描都请求 OSV.dev API。

## 部署结构

```text
联网更新任务 ──定期更新──> 本地漏洞数据库目录
                                  │
                         OSV-Scanner 离线扫描
```

单机部署时，更新任务和扫描任务使用同一台机器及同一个数据库目录即可。多台扫描机可将数据库目录同步到内网共享存储；更新完成后再发布给扫描机读取。

## 1. 安装 OSV-Scanner

在扫描机和更新机安装相同版本、匹配操作系统与 CPU 架构的 OSV-Scanner。记录版本号并在更新任务和扫描任务中保持一致。安装方式见 [OSV-Scanner 官方安装说明](https://google.github.io/osv-scanner/installation/)。

## 2. 配置本地数据库目录

设置 `OSV_SCANNER_LOCAL_DB_CACHE_DIRECTORY`，指向缓存根目录。以下以 Linux 为例：

```bash
export OSV_SCANNER_LOCAL_DB_CACHE_DIRECTORY=/var/lib/osv-scanner-cache
```

数据库文件会放在该目录下的 `osv-scanner/<ecosystem>/all.zip`，例如：

```text
/var/lib/osv-scanner-cache/
└── osv-scanner/
    ├── Go/all.zip
    ├── npm/all.zip
    └── PyPI/all.zip
```

只需准备项目实际使用的 ecosystem 数据库。

## 3. 定期联网更新数据库

在可以访问 OSV 数据库下载站的更新机上，为更新任务设置上述环境变量，并定期执行：

```bash
osv-scanner --offline-vulnerabilities --download-offline-databases ./project
```

将更新任务放入 cron、CI 定时流水线或其他调度器，例如每天运行一次。更新频率由漏洞时效要求和网络策略决定。更新成功后，多机部署再将整个缓存目录同步到扫描机或内网共享存储。

## 4. 使用本地数据库扫描

扫描任务设置相同的缓存根目录，使用 `--offline`：

```bash
export OSV_SCANNER_LOCAL_DB_CACHE_DIRECTORY=/var/lib/osv-scanner-cache
osv-scanner --offline scan source -r ./project \
  --format json --output-file ./result.json
```

`--offline` 让扫描器只使用本地数据库，不更新数据库，也不向外发送项目或依赖信息。扫描不再逐次调用 OSV.dev 查询漏洞；扫描机无需访问外网。若缺少项目所需的 ecosystem 数据库，扫描会报错，应先在更新机补齐并重新分发。

## 运维要点

- 监控数据库更新任务是否成功，并记录最近更新时间；离线扫描使用的是本地快照，数据新鲜度取决于更新频率。
- 多扫描机环境建议在更新完成后再发布数据库目录，避免扫描任务读到正在写入的文件。
- 保留最近一个可用快照，更新失败时可回滚。
- 该部署提供本地文件数据库给 OSV-Scanner 使用，不会启动一个供其他系统调用的 HTTP API 服务。若其他程序需要 API，应单独部署兼容 OSV API 的服务或直接使用 OSV-Scanner/OSV-SCALIBR 的接入方式。

官方参考：[OSV-Scanner Offline Mode](https://google.github.io/osv-scanner/usage/offline-mode/)。
