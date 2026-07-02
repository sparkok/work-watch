# Work-Watch 使用手册

> 依赖 [PilotDeck](https://github.com/OpenBMB/PilotDeck)，使用前请先启动 PilotDeck。

Work-Watch 是一个任务监工工具，配合 PilotDeck 使用。它将预先编写好的任务指令逐一提交给 PilotDeck 执行，自动跟踪进度，并在任务完成后向 PilotDeck 确认结果。

## 快速开始

双击 `work_watch.exe` 启动菜单模式，或从命令行运行：

```
work_watch <任务名>
```

首次启动时自动检测 PilotDeck 的配置（`~/.pilotdeck/pilotdeck.yaml`），自动生成 `config.yaml`，无需手动配置即可使用。

---

## 目录结构

```
work-watch/
├── config.yaml              # 全局配置（PilotDeck 连接信息）
├── work_watch.exe           # 可执行文件
├── tasks/                   # 任务目录
│   ├── 任务A/               # 每个任务一个子目录
│   │   ├── task.yaml        # 任务级配置（debug 模式、session_id）
│   │   ├── status           # YAML 状态文件（自动管理）
│   │   ├── jobs/            # 任务指令文件（*.txt）
│   │   │   ├── 001_xxx.txt
│   │   │   └── 002_yyy.txt
│   │   ├── logs/            # 执行日志
│   │   └── export/          # 导出结果
│   └── 任务B/
└── doc/
    └── 使用手册.md
```

### 全局配置 `config.yaml`

```yaml
pilotdeck:
  base_url: http://localhost:3001    # PilotDeck 服务地址
  api_key: <server-token>            # 从 ~/.pilotdeck/server-token 自动读取
  project_path: E:/study/ai/PilotDeck
```

> 启动时如果 `config.yaml` 不存在，会自动从 `~/.pilotdeck/pilotdeck.yaml` 和 `~/.pilotdeck/server-token` 读取配置并生成。

### 任务级配置 `tasks/<任务名>/task.yaml`

```yaml
debug: true                      # 调试模式
session_id: "abc123"             # 最后一次执行的会话 ID（自动管理）
```

---

## 任务文件

每个任务对应一个目录，`jobs/` 下放 `*.txt` 文件，每个文件是一条发给 PilotDeck 的指令。

文件按文件名**排序**依次执行（建议用数字前缀）：

```
jobs/
├── 001_需求分析.txt
├── 002_方案设计.txt
└── 003_代码实现.txt
```

文件内容是纯文本，直接作为消息发送给 PilotDeck。

---

## 使用方式

### 1. 菜单模式（无参数启动）

双击 `work_watch.exe` 或运行 `work_watch`，进入交互菜单：

```
========== Work-Watch 任务监工 ==========
 1. 配置    — 创建或修改任务配置
 2. 执行    — 执行任务 (异步)
 3. 结果导出 — 导出任务会话记录
 4. 状态    — 查看任务状态
 5. 退出
```

- **1 配置**：选择已有任务修改配置，或创建新任务
- **2 执行**：选择任务异步执行，不影响菜单操作
- **3 结果导出**：将任务的会话记录导出为 JSON / Markdown
- **4 状态**：查看所有任务的状态摘要

### 2. 命令行模式

| 命令 | 说明 |
|---|---|
| `work_watch` | 进入交互菜单 |
| `work_watch <任务名>` | 直接执行指定任务 |
| `work_watch config <任务名>` | 配置指定任务 |
| `work_watch export <任务名>` | 导出会话 JSON |
| `work_watch export <任务名> report` | 导出会话报告（Markdown） |
| `work_watch export <任务名> detail` | 导出详细会话（Markdown） |
| `work_watch status` | 查看所有任务状态 |

#### 示例

```bash
# 创建并执行一个新任务
work_watch my-task

# 查看所有任务状态
work_watch status

# 导出任务的会话报告
work_watch export my-task report
```

---

## 执行流程

当执行一个任务时，流程如下：

```
1. 检测配置文件
   ├── task.yaml 不存在 → 启动向导创建
   │   └── 向导完成后自动刷新 PilotDeck 连接配置
   ├── config.yaml 不存在 → 自动从 ~/.pilotdeck/ 初始化
   └── 检查 PilotDeck 服务是否在线

2. 逐条执行 job
   ┌── 读取下一个未完成的 *.txt 文件
   ├── 发送给 PilotDeck
   ├── 记录日志到 logs/
   ├── 标记为已完成（写入 status 文件）
   └── 重复直到所有 job 完成

3. 结果确认
   ├── 检查该任务是否已被确认过
   ├── 向 PilotDeck 发送确认消息
   ├── 解析回应中的成功/失败关键词
   ├── 不明确则重试（最多 3 次）
   └── 3 次都无法确定 → 按失败处理
```

### 断点续传

任务执行过程中如果中断（Ctrl+C、超时），已完成的 job **不会**重复执行。下次运行时会自动从下一个未完成的 job 继续。

要重新执行整个任务，删除 `status` 文件即可。

### 结果确认

所有 job 执行完毕后，会自动向 PilotDeck 询问任务结果：

```
正在向 PilotDeck 确认任务结果...
```

确认逻辑：
- PilotDeck 回复含 `成功` / `存在` / `success` → 判定为成功
- PilotDeck 回复含 `失败` / `不存在` / `fail` → 判定为失败
- 无法判断 → 重试（最多 3 次，间隔 3 秒）
- 3 次都无法确定 → 按失败处理

确认成功后会在 `status` 文件中记录 `confirmed: true`，下次运行不会再重复确认。

---

## 状态查看

运行 `work_watch status` 或菜单选"状态"，显示：

```
--- 任务状态 ---
  my-first-task (未配置)
  second-task [2/3 jobs]
  third-task [3/3 jobs, session: abc123]
```

| 状态 | 含义 |
|---|---|
| `(未配置)` | `task.yaml` 不存在，尚未配置 |
| `[2/3 jobs]` | 3 个 job 已完成 2 个 |
| `[3/3 jobs, session: xxx]` | 全部完成，关联会话 ID |
| `(配置错误: ...)` | 配置文件有问题 |

---

## 导出结果

完成任务后，可以将 PilotDeck 的会话记录导出为文件：

### JSON 导出（原始数据）

```bash
work_watch export my-task
```

输出到 `tasks/my-task/export/session-<id>-<时间戳>.json`

### Report 导出（摘要报告）

```bash
work_watch export my-task report
```

生成 Markdown 格式的摘要报告，包含：
- 用户请求列表
- 助手回复摘要
- 工具调用记录
- 错误信息

### Detail 导出（完整记录）

```bash
work_watch export my-task detail
```

生成包含完整消息内容的 Markdown 文档。

---

## 配置文件详解

### 全局配置（`config.yaml`）

存储 PilotDeck 连接信息，由以下方式产生（优先级从高到低）：

1. **环境变量**：`HOST` + `PORT` 组成 base_url，`API_KEY` 作为 api_key，`PROJECT_PATH` 作为 project_path
2. **自动发现**：启动时从 `~/.pilotdeck/pilotdeck.yaml` 读取端口 → `http://localhost:<port>`，从 `~/.pilotdeck/server-token` 读取 API key
3. **交互向导**：在菜单中选"配置"，按提示输入
4. **默认值**：`http://localhost:3001`

### 环境变量

| 变量 | 说明 |
|---|---|
| `HOST` | PilotDeck 服务主机 |
| `PORT` | PilotDeck 服务端口 |
| `API_KEY` | API 密钥 |
| `PROJECT_PATH` | PilotDeck 项目路径 |
| `PILOT_HOME` | PilotDeck 配置目录（默认 `~/.pilotdeck`） |

---

## 常见问题

### Q: 启动时提示"PilotDeck 服务未启动"

确保 PilotDeck 服务正在运行。检查 `config.yaml` 中的 `base_url` 是否正确，默认是 `http://localhost:3001`。

### Q: 报错 "Invalid or inactive API key"

API key 过期或不匹配。如果删除了 `task.yaml` 重新配置，会自动从 `~/.pilotdeck/server-token` 刷新 API key。也可以手动更新 `config.yaml` 中的 `api_key` 字段。

### Q: 想重新执行整个任务

删除 `tasks/<任务名>/status` 文件即可，下次运行时会从第一个 job 重新开始。

### Q: 任务状态显示"未配置"

`task.yaml` 不存在。执行该任务时向导会自动创建，也可以手动运行 `work_watch config <任务名>` 来配置。

### Q: 任务执行到一半中断了

已完成的部分已保存到 `status` 文件中。重新运行会从断点继续，不会重复执行已完成的 job。
