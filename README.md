# 账号生成器 (Account Generator)

一键批量生成测试账号数据，支持 **Web UI** 和 **命令行** 两种操作方式。

生成的账号包含姓名、邮箱、用户名、密码、手机号，自动保存到本地文件，并内置 HTTP 接口可供其他程序调用。

---

## 功能特性

- **双模式操作** — 浏览器 Web UI 界面点按生成，或终端命令行按 Enter 生成
- **唯一用户名** — 随机字母部分基于文件去重，确保每次生成均不重复
- **自动保存** — 按日期追加到 `YYYYMMDD.txt`，同时输出 `latest.json` 供外部读取
- **一键复制** — Web UI 每个字段独立复制按钮，点击即复制到剪贴板
- **HTTP API** — 内置 REST 接口，可供其他工具或脚本集成调用
- **跨平台** — 纯 Go 编译，单文件无依赖，Windows / macOS / Linux 均可运行
- **端口自适应** — 端口被占用时自动累加尝试，无需手动更换
- **可配置随机长度** — 支持 `--rand-len` 参数调整随机字母位数

---

## 快速开始

### 方式一：下载预编译二进制

从 [Releases](../../releases) 页面下载对应平台的可执行文件，直接运行：

```bash
# Windows
account_gen.exe

# macOS / Linux
./account_gen
```

### 方式二：从源码编译

```bash
# 克隆仓库
git clone https://github.com/<your-username>/account-generator.git
cd account-generator

# 编译
go build -o account_gen.exe .

# 运行
./account_gen.exe
```

> 需要 Go 1.21+，查看 [构建指南](#构建说明) 了解更多。

---

## 使用指南

### Web UI 模式（默认）

启动后自动在浏览器中打开 UI 界面：

```bash
account_gen.exe
```

终端将显示：
```
> 输出目录: ./output
> 随机字母长度: 13
> 打开浏览器访问 http://localhost:8080 使用UI界面
```

在浏览器中：
1. 点击 **生成账号** 按钮
2. 账号数据即时显示在卡片中
3. 点击字段旁的 **复制** 按钮复制单个字段
4. 顶部徽章显示已生成总数
5. 底部显示生成时间戳

### CLI 模式

启动后在终端中按 Enter 生成，输入 `q` 退出：

```bash
# 默认模式（同时启动 Web UI，终端按 Enter 也能生成）
account_gen.exe

# 纯 CLI 模式（不启动 Web UI）
account_gen.exe --cli-only

# CLI 模式下指定输出目录
account_gen.exe --cli-only ./myoutput
```

```
> testbit
> 名：testbit
> 邮箱：testbit260529220503@uuf.me
> 用户名：testbitthzzdx
> 密码：testbit#P220503
> 手机号：260529220503
> ----------------------------------------
```

---

## 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `[输出目录]` | 指定账号文件保存目录（首个非 `--` 参数） | `./output` |
| `--port PORT` | 指定 Web UI 的 HTTP 端口 | `8080` |
| `--rand-len N` | 指定随机字母部分的长度 | `13` |
| `--cli-only` | 仅启动命令行模式，不启动 Web UI | — |
| `-h` / `--help` | 显示帮助信息 | — |

### 使用示例

```bash
# 基本用法，默认端口 8080，输出到 ./output
account_gen.exe

# 指定输出目录
account_gen.exe ./my_accounts

# 指定端口
account_gen.exe --port 9090

# 调整随机字母长度为 8 位
account_gen.exe --rand-len 8

# 组合使用
account_gen.exe ./data --port 9090 --rand-len 6

# 仅命令行模式，8 位随机字母
account_gen.exe --cli-only --rand-len 8
```

---

## 输出说明

### 文本文件 `YYYYMMDD.txt`

按日期追加，每条记录格式：

```
姓：testbit
名：testbit
邮箱：testbit260529220503@uuf.me
用户名：testbitthzzdx
密码：testbit#P220503
手机号：260529220503
```

### JSON 文件 `latest.json`

始终保存最新生成的一条账号，方便外部程序读取：

```json
{
  "created_at": "2026-05-29 22:05:03",
  "email": "testbit260529220503@uuf.me",
  "firstName": "testbit",
  "first_name": "testbit",
  "lastName": "testbit",
  "last_name": "testbit",
  "mobile_number": "260529220503",
  "name": "testbit",
  "password": "testbit#P220503",
  "phone": "260529220503",
  "username": "testbitthzzdx"
}
```

### 去重文件 `unique_names.txt`

记录所有已使用的随机字母串，防止重复，自动创建在可执行文件所在目录。

---

## 生成规则

账号各字段由当前时间 + 随机字母组成：

| 字段 | 规则 | 示例 |
|------|------|------|
| 姓 / 名 | 固定前缀 | `testbit` |
| 邮箱 | `testbit` + 日期时间数字 + `@uuf.me` | `testbit260529220503@uuf.me` |
| 用户名 | `testbit` + N 位唯一随机小写字母 | `testbitthzzdx` |
| 密码 | `testbit#P` + 时分秒 | `testbit#P220503` |
| 手机号 | 日期 + 时分秒（纯数字） | `260529220503` |

- 日期格式：`YYMMDD`（示例：`260529` = 2026-05-29）
- 时间格式：`HHmmss`（示例：`220503` = 22:05:03）
- 随机字母：**a–z** 小写，长度默认为 13（可通过 `--rand-len` 调整）

---

## API 接口

Web UI 模式下内置以下 HTTP 接口，可供其他工具集成：

### `GET /api/generate`

生成一条新账号并返回 JSON。

```bash
curl http://localhost:8080/api/generate
```

### `GET /api/latest`

获取最新生成的一条账号 JSON。

```bash
curl http://localhost:8080/api/latest
```

### `GET /api/history`

获取已生成的文件列表。

```bash
curl http://localhost:8080/api/history
```

```json
{
  "files": ["20260529.txt", "20260530.txt"],
  "total": 2
}
```

所有 API 均设置了 `Access-Control-Allow-Origin: *`，支持跨域调用。

---

## 项目结构

```
account-generator/
├── main.go              # 主程序（CLI + Web UI + API，含嵌入式 HTML）
├── go.mod               # Go 模块定义
├── go.sum               # 依赖校验文件（使用标准库时无需）
├── README.md            # 本文档
├── .gitignore           # Git 忽略规则
├── output/              # 账号输出目录（自动创建）
│   ├── 20260529.txt     # 按日期归集的账号文本
│   └── latest.json      # 最新一条账号 JSON
└── unique_names.txt     # 已使用的随机字母库（自动创建）
```

> `output/` 和 `unique_names.txt` 由程序自动生成，无需手动创建。

---

## 构建说明

### 前置要求

- [Go](https://go.dev/dl/) 1.21 或更高版本

### 基本构建

```bash
go build -o account_gen .
```

### 跨平台构建

```bash
# Windows (64位)
GOOS=windows GOARCH=amd64 go build -o account_gen_windows_amd64.exe .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o account_gen_darwin_amd64 .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o account_gen_darwin_arm64 .

# Linux (64位)
GOOS=linux GOARCH=amd64 go build -o account_gen_linux_amd64 .
```

### 体积优化

```bash
# 去掉调试信息，减小体积
go build -ldflags="-s -w" -o account_gen.exe .
```

### 验证

```bash
# 代码静态检查
go vet ./...

# 运行（按 Ctrl+C 停止）
./account_gen
```

---

## 常见问题

**Q：端口被占用怎么办？**

程序会自动检测并尝试下一个端口（8080 → 8081 → 8082 ...）。你也可以用 `--port` 手动指定。

**Q：随机字母用完了怎么办？**

26 个小写字母，13 位长度的组合空间为 `26^13 ≈ 2.48×10^18`，几乎不会用完。如果调短了长度且实际耗尽，程序会进入死循环等待新的可用组合，此时删除 `unique_names.txt` 即可重置。

**Q：如何修改邮箱后缀或密码前缀？**

目前为编译时常量，需修改源码中 `emailSuffix` 和 `passwordPre` 后重新编译。

**Q：生成的账号有什么实际用途？**

适用于需要批量注册测试账号的场景，如接口测试、压力测试、演示环境数据准备等。

---

## 技术栈

- **语言**：Go 1.26
- **Web UI**：纯内联 HTML + CSS + JavaScript（无外部依赖）
- **随机数**：`crypto/rand` 密码学安全随机数
- **HTTP**：Go 标准库 `net/http`

---

## 许可证

[MIT](LICENSE)