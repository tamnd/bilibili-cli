# bili

[![CI](https://github.com/tamnd/bilibili-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/tamnd/bilibili-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/tamnd/bilibili-cli)](https://github.com/tamnd/bilibili-cli/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/tamnd/bilibili-cli.svg)](https://pkg.go.dev/github.com/tamnd/bilibili-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/tamnd/bilibili-cli)](https://goreportcard.com/report/github.com/tamnd/bilibili-cli)
[![License](https://img.shields.io/github/license/tamnd/bilibili-cli)](./LICENSE)

[English](README.md) · [简体中文](README.zh-CN.md)

一个用于 [bilibili.com](https://www.bilibili.com) 的命令行工具。`bili` 能把任意视频、用户、评论、弹幕、动态、直播间、番剧、音频、专栏或收藏夹解析成干净的结构化记录，并通过 BBDown 下载音频。单个纯 Go 二进制，无需 API key，访问任何公开内容都无需登录。

[安装](#安装) • [命令](#命令) • [用法](#用法) • [工作原理](#工作原理)

![bili 在命令行中搜索 bilibili 并读取视频记录](docs/static/demo.gif)

它通过普通 HTTPS 与 bilibili 的公开 Web 接口通信：WBI 签名、匿名 `buvid` 会话初始化、`{code, message, data}` 响应包络拆解、av/BV 号互转，以及 protobuf 弹幕解码，全都替你处理好了。Cookie 是可选的：传入 `--cookie`，`bili` 就能拿到与你已登录浏览器相同的数据。音频下载交给 `BBDown`，当请求的格式与它输出的格式不一致时，由 ffmpeg 完成转码。

`bili` 是一个独立工具，与哔哩哔哩官方无关。

## 安装

```bash
go install github.com/tamnd/bilibili-cli/cmd/bili@latest
```

也可以从 [releases](https://github.com/tamnd/bilibili-cli/releases) 下载预编译二进制，或运行容器镜像：

```bash
docker run --rm ghcr.io/tamnd/bili:latest search 'lofi' -n 10
```

内置 shell 补全：`bili completion bash|zsh|fish|powershell`。

## 命令

| 命令 | 读取内容 |
| --- | --- |
| `bili video <id\|url>...` | 一个或多个视频；完整元数据 |
| `bili download <id\|url>` | 音频，经由 BBDown；`--format`、`--parts`、`--output-dir` |
| `bili related <id\|url>` | 某个视频的相关推荐 |
| `bili streams <id\|url>` | 视频分 P 的可播放流地址 |
| `bili danmaku <id\|url>` | 视频分 P 的弹幕；`--page` |
| `bili comments <id\|url>` | 视频、专栏、音频或动态下的全部评论与回复 |
| `bili user <mid\|url>` | 创作者的资料、投稿目录、统计或动态；`--videos`、`--dynamics` |
| `bili dynamic <id\|url>` | 单条动态的完整内容 |
| `bili dynamics <mid\|url>` | 某个用户的全部动态流 |
| `bili favorite <ml\|url>` | 某个收藏夹内的视频 |
| `bili favorites <mid\|url>` | 某个用户的收藏夹列表 |
| `bili bangumi <ss\|ep\|md\|url>` | 某一季动画或影视及其全部剧集 |
| `bili audio <au\|url>` | 某条音频的元数据与统计 |
| `bili article <cv\|url>` | 某篇专栏的元数据与正文 |
| `bili live <room\|url>` | 直播间信息，或按分区浏览直播间 |
| `bili search <query>` | 搜索视频、用户、番剧、直播间或专栏；`--type` |
| `bili suggest <term>` | 搜索联想词 |
| `bili trending` | 当前热搜词 |
| `bili popular` | 热门内容流，或某一期每周必看 |
| `bili rank` | 排行榜，可指定单个分区 |
| `bili id <thing>` | 识别并规范化任意 id 或 URL |
| `bili discover <id\|url>...` | 从视频或创作者出发，对关系图做广度优先遍历；`--follow`、`--depth`、`--fanout` |
| `bili crawl <id\|url>...` | 从种子 id 出发抓取相互关联的记录，写入 JSONL 文件 |
| `bili nav` | 登录状态与当前 WBI 密钥（调试用） |
| `bili verify --live` | 对照线上 API 重新测定各接口的能力矩阵 |
| `bili config show` | 解析后的配置与路径 |
| `bili cache path\|stat\|clear` | 查看或清除磁盘缓存 |
| `bili version` | 打印版本、提交号与构建日期 |

完整参考与指南见 [bilibili-cli.tamnd.com](https://bilibili-cli.tamnd.com)。

## 用法

```bash
bili video BV17x411w7KC                    # 完整的视频元数据
bili download BV17x411w7KC                 # 它的音频，m4a 格式
bili comments BV17x411w7KC -n 50           # 热门评论及其回复
bili danmaku BV17x411w7KC                  # 第一个分 P 的弹幕
bili search 'lofi' -n 20                   # 搜索视频
bili user 122541                           # 创作者的资料
bili bangumi ss12548                       # 一季动画
bili rank --tid 129                        # 舞蹈区排行榜
```

记录可以输出为表格（终端上的默认格式）、列表、markdown、JSON、JSONL、CSV、TSV、url 或原始格式。在真彩终端上，表格使用圆角边框和彩色表头；JSON 与 JSONL 同样带有语法高亮：

```bash
bili video BV17x411w7KC --fields bvid,title,view_count,like_count -o table
bili search 'lofi' -o jsonl | jq '{bvid, title, view_count}'
bili search 'lofi' -o url
bili user 122541 --videos -o jsonl
bili comments BV17x411w7KC --replies -o jsonl > comments.jsonl
```

### 下载音频

`bili download` 包装了 [BBDown](https://github.com/nilaoda/BBDown)，这个工具已经解决了其中困难的部分：DASH manifest、流选择、多分 P 处理以及合并。你粘贴的任何内容都会先被解析成 bvid，因此 URL、裸 av 号和 bvid 的用法与其它命令完全一致。

```bash
bili download BV17x411w7KC                              # m4a，直接由 BBDown 输出
bili download BV17x411w7KC --parts 1-3 --format mp3     # 用 ffmpeg 转码
bili download BV17x411w7KC --format flac --output-dir ~/Music
```

`m4a` 是 BBDown 直接输出的格式，不涉及第二个进程。`mp3`、`flac` 和 `wav` 会在下载完成后转码一次，这需要 PATH 中有 `ffmpeg`。`--quality` 设置 mp3 的码率预设，对 `flac` 和 `wav` 会被忽略——无损格式没有它可作用的地方。

两个二进制都没有内置。二者都会在 PATH 中查找，`--bbdown-bin` 和 `--ffmpeg-bin` 可以指向放在别处的副本；缺失的那个会在传输开始之前就报出来，而不是等传输结束后。这个命令只下载音频，不下载视频，也不解密任何内容。

抓取一次搜索结果，并为每个结果拉取评论和 UP 主资料：

```bash
bili search 'lofi' -n 20 -o url \
  | bili crawl - --out ./data --comments --user
```

### 全局参数

```
-o, --output       list|table|markdown|json|jsonl|csv|tsv|url|raw   （自动：TTY 上用 table，管道中为 jsonl）
    --fields       要保留的列，逗号分隔，按顺序
    --no-header    省略表头行
    --template     对每条记录应用的 Go text/template
-n, --limit        最大记录数（0 = 不限）
    --cookie       cookie 请求头（SESSDATA=...; ...）
    --cookie-file  Netscape cookie 文件路径
    --lang         本地化字段的 locale（默认 zh-CN）
-q, --quiet        抑制进度输出
    --color        auto|always|never
    --rate         请求之间的最小间隔（默认 350ms）
    --timeout      单次请求超时（默认 30s）
    --retries      遇到 429 与 5xx 时的重试次数（默认 4；风控拒绝永不重试）
-j, --workers      扇出类命令的并发数（默认 4）
    --no-cache     绕过磁盘缓存
    --cache-ttl    缓存有效期（默认 1h）
    --dry-run      打印将要发出的请求
    --proxy        HTTP 或 SOCKS 代理 URL
```

## 工作原理

bilibili 的公开 API 建立在若干共通的约定之上。`bili` 把它们处理掉了，你无需操心：

**WBI 签名。** 部分接口会拒绝未签名的请求。`bili` 从 nav 接口获取当前的 WBI 密钥对，推导出 mixin key，然后用 `w_rid` 和 `wts` 为每次调用签名。

哪些接口需要签名是一个实测出来的事实，而不是能从 URL 上读出来的规则：`x/web-interface/wbi/search/type` 的路径里带着 `wbi`，却完全不需要签名。`bili/matrix.go` 记录了对每个接口实测出的需求，`bili verify --live` 会重新测定。

**匿名会话。** `bili` 首次使用时会激活一对新的 `buvid3`/`buvid4`，让那些期望浏览器会话的接口能正常应答。这本身就是一道独立的门槛，与签名无关：`bili rank` 不发送签名也能工作，但缺少 buvid cookie 时会被拒绝。

**响应包络。** 响应以 `{code, message, ttl, data}` 的形式到达（番剧接口为 `result`）。`bili` 在解码 payload 的任何一个字节之前，先把每个响应归入七种状态之一，然后才拆开包络。

多出这一步，是因为在这个 API 上 `code: 0` 并不代表成功。有两个接口会返回成功码、成功消息，内容却是空的，而一个只读 code 就开始解码的客户端，无法把它和用户收藏夹确实为空区分开来。`bili` 能区分，因为它记录了哪些接口在应答时会携带 payload，所以 `bili favorites` 会指明这是一次拒绝，而不是打印 `[]`。风控的 HTML 拦截同样会被识别为拒绝，而不是冒泡成 JSON 解析错误；任何拒绝都不会被写入缓存。

同一条规则也适用于逐页读取。分页内容流的结束依据是服务器说 `has_more: false`，而不是某一页碰巧没有条目，因此在遍历途中被拒绝的一页会被报告出来，而不是悄悄结束列表。收藏夹读取得到的信号还要更强：接口会把收藏夹自己的 `media_count` 与内容一起返回，所以一个声称有 145 条却一条都没发来的收藏夹，是按算术判定出的拒绝，而不是靠猜。

**来源信息（provenance）。** 来自 `bili` 的一条记录很少来自单个响应。`bili user` 的一行背后是四个请求：`acc/info` 提供身份信息，粉丝数、投稿数、播放数和点赞数来自另外三个接口，各有各的门槛，其中任何一个都可能被拒绝，而其余仍然正常应答。因此每条记录都带有一个 `envelope`，说明是哪个接口应答的、请求是否签名、响应被归入哪种状态、发生在什么时候、返回了多少字节。

真正划算的字段是 `envelope.missed`。被拒绝的计数会从记录中整个省略，而不是打印成 0，并在那里注明是哪个接口、哪种状态挡住了它。确实为 0 的计数仍然是 0。在表格或 csv 中，被扣下的计数是一个空单元格——当下游要对这一列做算术时，这个差别很重要。

```console
$ bili user 946974 -o jsonl | jq '{video_count, total_view, missed: .envelope.missed}'
{
  "video_count": 924,
  "total_view": null,
  "missed": {
    "total_like": "x/space/upstat refused_silent: code 0 with no payload",
    "total_view": "x/space/upstat refused_silent: code 0 with no payload"
  }
}
```

再运行一次，`video_count` 可能也会加入这个列表：`arc/search` 对匿名调用者会间歇性地触发风控，而这正是当拒绝和 0 看起来一样时，一条记录无法告诉你的东西。

包络不会出现在表格和 csv 中，因为来源信息值得存在于每条记录上，却不值得占用任何一列。需要查看时，用 `--fields envelope` 按名索取。

**ID。** 视频同时拥有 `avNNN` 数字和 `BV` 字符串两种标识。`bili` 可以在两者之间互转，会跟随 `b23.tv` 短链，并可用 `bili id` 对你粘贴的任何 id 或 URL 进行分类。

**弹幕。** 弹幕以 protobuf 分段流的形式下发。`bili` 将其解码为每条评论一条记录，包含时间戳、模式、颜色和文本。

除了 CLI，还附带两个可导入的包：

| 包 | 作用 |
| --- | --- |
| `pkg/bvconv` | 在 `avNNN` 数字与 `BV` 字符串之间互转 |
| `pkg/dmproto` | 解码 protobuf 弹幕分段流 |

## 退出码

拒绝和空结果是两种不同的结果，因此使用不同的退出码。脚本无需读取 stderr 就能据此采取行动。

| 退出码 | 含义 |
| --- | --- |
| 0 | 完成了要求它做的事 |
| 1 | 本工具无法完成要求，且说不清原因：无法理解的参数或选项、无法归类的响应，或中途被中断的运行 |
| 2 | 风控拒绝了请求，表现为 `-352` 或 HTTP 412。重试不会解除，带上已登录的 cookie 通常可以 |
| 3 | 请求成功，且确实没有任何内容可返回 |
| 4 | API 返回了成功码却不携带 payload，而该接口总是会携带。这是披着成功码外衣的拒绝 |
| 5 | 网络故障、超时，或重试耗尽后仍然存在的 5xx |
| 6 | 被限流：`-509` 或重试耗尽后仍然存在的 429。这一种等一等就会解除 |
| 7 | 未找到，表现为 `-404` 或某个接口自用的私有常量 |

```console
$ bili audio au1 >/dev/null; echo $?
7
$ bili favorites 946974 >/dev/null; echo $?
4
$ bili video BV1xx411c7mD --tags >/dev/null; echo $?
3
```

对于给定多个标识符的命令，退出码描述的是整次运行，而不是单个目标。中途的失败不会阻止其余目标继续，计数会写到 stderr，只有当某个状态覆盖了所有目标时，它才会成为整次运行的退出码。五百个收藏夹里有一个读取被拒绝，不等于整次运行被拒绝。

`bili discover` 在图上而非列表上遵循同样的规则。被门槛拦住的边是 stderr 上的一条注记，遍历会继续；结束时按类别统计这些注记，只有一次遍历在种子之外什么都没触及时，才会以拒绝对应的退出码退出。

## 开发

```
cmd/bili/    精简的 main 入口
cli/         cobra 命令与输出渲染
bili/        HTTP 客户端、WBI 签名、会话初始化、id 解析、数据模型
pkg/bvconv/  av/BV id 互转（无依赖）
pkg/dmproto/ protobuf 弹幕解码器（无依赖）
docs/        文档站点（Hugo，tago-doks 主题）
```

```bash
make build   # ./bin/bili
make test    # go test ./...
make vet     # go vet ./...
make smoke   # 构建 + 线上冒烟脚本
```

需要 Go 1.26+。

测试套件完全离线运行，而且将始终如此：存储下来的抓包直接进入分类器，其周边的策略（缓存、重试）则运行在 stub transport 之上。这一点是被验证的而非仅仅声称的——每个测试二进制都在 `sandbox-exec` 下、禁用网络运行，因为一个会访问网络的测试在有网络的机器上照样能通过。唯一无法离线验证的，是 bilibili 是否仍按本仓库所相信的方式运行，而这正是 `bili verify --live` 的用途。它不属于 `go test`，因为会访问网络的测试套件在火车上会失败。`drift` 工作流每周运行一次，当某一行发生变化时就开一个 issue。

## 发布

先在 [CHANGELOG.md](CHANGELOG.md) 中写好新版本的章节，然后推送一个版本 tag，GitHub Actions 会运行 GoReleaser：

```bash
git tag -a v0.3.0 -m "v0.3.0"
git push --tags
```

发布说明就是 changelog 中的那一节，而不是一堆提交标题。没有对应章节的 tag 会在发布任何内容之前就让发布失败，因为提交列表说的是做了什么，而从不说明它对已经装了上一版本的人意味着什么。

镜像 tag 不带 `v` 前缀（`ghcr.io/tamnd/bili:0.3.0`）。

## 许可证

Apache-2.0。见 [LICENSE](LICENSE)。

`bili` 是一个独立的客户端。请在 bilibili 的服务条款范围内负责任地访问公开数据。
