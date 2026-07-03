# work-watch 快速引导

> 依赖 [PilotDeck](https://github.com/OpenBMB/PilotDeck)，使用前请先启动 PilotDeck。
>
**一个命令，批量驱动 PilotDeck AI 完成你的工作流。**

电脑高手帮你配好 PilotDeck 之后，你只需要做一件事：**写任务文件，然后运行。**

## 三步搞定

### 1. 写任务文件

在 `tasks/<任务名>/jobs/` 目录下新建 `*.txt` 文件，每个文件写一条你想让 AI 做的事：

```
tasks/my-task/jobs/
├── 001_需求分析.txt
├── 002_设计方案.txt
└── 003_代码实现.txt
```

文件名用数字前缀控制顺序，内容就是给 AI 的指令，纯文本即可。

### 2. 运行

```
work-watch my-task
```

AI 会按顺序逐个处理你的 txt 文件，实时显示进度。

### 3. 收结果

任务跑完后导出会话记录：

```
work-watch export my-task        # JSON 格式
work-watch export my-task report # 报告格式
```

## 为什么用它

- **你只管写 txt** — 把要干的活写进文件，剩下的交给 work-watch
- **无需配置** — PilotDeck 连接由高手一次配好，你开箱即用
- **进度一目了然** — 实时显示每个任务的执行状态
- **中断可续** — 断了接着跑，不重复干活
- **菜单式操作** — 记不住命令？直接双击 `work-watch.exe` 进菜单

---

**把你不想手动干的活，一个个写成 txt，让 AI 去跑。**

