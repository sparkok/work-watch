# work-watch 快速引导

> 依赖 [PilotDeck](https://github.com/OpenBMB/PilotDeck)，使用前请先启动 PilotDeck。
>
> 典型场景：由电脑高手配置好 PilotDeck，其他用户只需使用 work-watch 即可。

**一个命令，批量驱动 PilotDeck AI 完成你的工作流。**

手动给 AI 发请求、等回复、再发下一个……累不累？把你要干的活写成任务列表，`work-watch` 一个接一个帮你提交，你只需要收结果。

## 三步搞定

1. **配任务** — 列出要执行的 jobs
2. **跑一下** — `work-watch
3. **收结果** — 喝杯咖啡，回来看会话记录

## 为什么用它

- **零配置** — 自动发现 PilotDeck 连接信息，下载就能用，不用填任何东西
- **一次配，反复跑** — 任务配置写一次，重置会话就能重跑，适合迭代调试
- **进度一目了然** — 实时显示每个 job 的提交状态和会话 ID，跑没跑完心里有数
- **导出随心** — JSON 二次处理、Markdown 报告归档、完整交互记录复盘，三种格式任选
- **菜单式操作** — 记不住命令？直接运行 `work-watch` 进交互菜单，选着用
- **中断可续** — 会话状态自动保存，断了接上，不必从头来

## 快速开始

```bash
go build -o work-watch.exe
work-watch                  # 进菜单配置并运行
work-watch myTask           # 直接跑指定任务
work-watch export myTask    # 导出会话记录
```

---

**把你不想手动干的活，写成任务列表，让 AI 去跑。**
