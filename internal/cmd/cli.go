package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"work-watch/internal/jobs"
	"work-watch/internal/pilotdeck"
	"work-watch/internal/task"
)
func startupHealthCheck() {
	baseURL := task.DefaultBaseURL
	if h, p := os.Getenv("HOST"), os.Getenv("PORT"); h != "" && p != "" {
		baseURL = fmt.Sprintf("http://%s:%s", h, p)
	} else if task.HasGlobalConfig() {
		if cfg, err := task.LoadConfig(""); err == nil {
			baseURL = cfg.PilotDeck.BaseURL
		}
	}
	if checkServerHealth(baseURL) {
		fmt.Printf("✓ PilotDeck %s is healthy\n", baseURL)
	} else {
		fmt.Fprintf(os.Stderr, "✗ PilotDeck %s is unreachable\n", baseURL)
	}
}

// Run parses args and dispatches to the right mode.
func Run(args []string) int {
	// Determine app root directory (where work-watch.exe lives)
	if exe, err := os.Executable(); err == nil {
		task.AppDir = filepath.Dir(exe)
	} else if cwd, err := os.Getwd(); err == nil {
		task.AppDir = cwd
	}

	startupHealthCheck()

	if len(args) == 0 {
		return runMenuMode()
	}

	subcommand := args[0]

	if subcommand == "config" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: work-watch config <taskName>")
			return 1
		}
		return runConfigMode(args[1])
	}

	if subcommand == "export" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: work-watch export <taskName> [json|report|detail]")
			return 1
		}
		format := "json"
		if len(args) >= 3 {
			switch args[2] {
			case "report", "detail":
				format = args[2]
			}
		}
		return runExportMode(args[1], format)
	}

	if subcommand == "status" {
		return runStatusMode()
	}

	// Otherwise treat as task name (direct run)
	return runTaskMode(subcommand)
}
// ===================== Interactive Menu =====================

func runMenuMode() int {
	var asyncWg sync.WaitGroup

	for {
		fmt.Println("\n========== Work-Watch 任务监工 ==========")
		fmt.Println(" 1. 配置    — 创建或修改任务配置")
		fmt.Println(" 2. 执行    — 执行任务 (异步)")
		fmt.Println(" 3. 结果导出 — 导出任务会话记录")
		fmt.Println(" 4. 状态    — 查看任务状态")
		fmt.Println(" 5. 退出")
		fmt.Print("\n请选择 (1-5): ")

		input := task.ReadLine()

		switch input {
		case "1", "配置":
			menuConfig()
		case "2", "执行":
			menuRun(&asyncWg)
		case "3", "结果导出":
			menuExport()
		case "4", "状态":
			menuStatus()
		case "5", "退出":
			fmt.Println("等待异步任务完成...")
			asyncWg.Wait()
			fmt.Println("再见!")
			return 0
		default:
			fmt.Println("无效选择，请重新输入。")
		}
	}
}
func menuConfig() {
	tasks, _ := task.ListTasks()
	if len(tasks) > 0 {
		fmt.Println("\n--- 已有任务 ---")
		for i, t := range tasks {
			fmt.Printf("  %d. %s\n", i+1, t)
		}
		fmt.Printf("  %d. 创建新任务\n", len(tasks)+1)
		fmt.Print("\n选择任务进行配置, 或创建新任务: ")

		input := task.ReadLine()

		if n, err := strconv.Atoi(input); err == nil {
			if n >= 1 && n <= len(tasks) {
				runConfigMode(tasks[n-1])
				return
			} else if n == len(tasks)+1 {
				// fall through to create
			}
		}
		for _, t := range tasks {
			if strings.EqualFold(t, input) {
				runConfigMode(t)
				return
			}
		}
	}

	fmt.Print("\n输入新任务名称: ")
	name := task.ReadLine()
	if name != "" {
		runTaskMode(name)
	}
}
func menuRun(wg *sync.WaitGroup) {
	tasks, err := task.ListTasks()
	if err != nil || len(tasks) == 0 {
		fmt.Println("没有可用的任务。请先创建任务。")
		return
	}

	fmt.Println("\n--- 选择要执行的任务 ---")
	for i, t := range tasks {
		statusLine := taskStatusLine(t)
		fmt.Printf("  %d. %s %s\n", i+1, t, statusLine)
	}

	fmt.Print("\n选择任务: ")
	taskName := chooseTask(tasks)
	if taskName == "" {
		fmt.Println("无效选择。")
		return
	}

	taskDir := task.TaskDir(taskName)
	cfg, err := task.LoadConfig(taskDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		return
	}
	if !checkServerHealth(cfg.PilotDeck.BaseURL) {
		fmt.Fprintf(os.Stderr, "PilotDeck 服务未启动 (%s)\n", cfg.PilotDeck.BaseURL)
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Printf("\n▶ 开始异步执行任务: %s\n", taskName)

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\n任务被中断。已完成进度已保存。")
			cancel()
		}()

		start := time.Now()
		err := jobs.RunTask(ctx, &jobs.RunOptions{
			TaskDir: taskDir,
			Debug:   cfg.Debug,
			Cfg:     cfg,
			OnJobDone: func(jobName string, resp *pilotdeck.AgentResponse) {
				fmt.Printf("  ✓ %s (session: %s)\n", jobName, resp.SessionID)
			},
		})
		elapsed := time.Since(start)

		if err != nil {
			if ctx.Err() != nil {
				fmt.Printf("⏹ 任务 %s 已停止 (%s)\n", taskName, elapsed.Round(time.Second))
			} else {
				fmt.Fprintf(os.Stderr, "✗ 任务 %s 失败: %v\n", taskName, err)
			}
		} else {
			fmt.Printf("✓ 任务 %s 完成 (%s)\n", taskName, elapsed.Round(time.Second))
		}
	}()
}

func chooseTask(tasks []string) string {
	input := task.ReadLine()
	if n, err := strconv.Atoi(input); err == nil && n >= 1 && n <= len(tasks) {
		return tasks[n-1]
	}
	for _, t := range tasks {
		if strings.EqualFold(t, input) {
			return t
		}
	}
	return ""
}

func menuExport() {
	tasks, err := task.ListTasks()
	if err != nil || len(tasks) == 0 {
		fmt.Println("没有可用的任务。")
		return
	}

	fmt.Println("\n--- 选择要导出结果的任务 ---")
	for i, t := range tasks {
		taskDir := task.TaskDir(t)
		cfg, _ := task.LoadConfig(taskDir)
		sid := ""
		if cfg != nil && cfg.SessionID != "" {
			sid = " (session: " + cfg.SessionID + ")"
		}
		fmt.Printf("  %d. %s%s\n", i+1, t, sid)
	}

	fmt.Print("\n选择任务: ")
	taskName := chooseTask(tasks)
	if taskName == "" {
		fmt.Println("无效选择。")
		return
	}

	fmt.Println("\n选择导出格式:")
	fmt.Println("  1. JSON       — 机器可读，可用于分析")
	fmt.Println("  2. 报告       — 简明报告，辅助决策")
	fmt.Println("  3. 详细交互   — 完整交互过程，方便阅读")
	fmt.Print("\n请选择 (1-3): ")

	format := "json"
	switch task.ReadLine() {
	case "2", "报告":
		format = "report"
	case "3", "详细交互":
		format = "detail"
	}

	runExportMode(taskName, format)
}

// menuStatus: show all task statuses
func menuStatus() {
	runStatusMode()
}

// ===================== Task Execution =====================

func runTaskMode(taskName string) int {
	taskDir := task.TaskDir(taskName)

	// Check if task config exists — if not, enter wizard
	if _, err := os.Stat(task.ConfigPath(taskDir)); os.IsNotExist(err) {
		fmt.Printf("Task %q needs configuration. Starting wizard...\n", taskName)
		if err := task.CreateTaskWizard(taskName); err != nil {
			fmt.Fprintf(os.Stderr, "Wizard error: %v\n", err)
			return 1
		}
		taskDir = task.TaskDir(taskName)
	}

	cfg, err := task.LoadConfig(taskDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		return 1
	}

	if !checkServerHealth(cfg.PilotDeck.BaseURL) {
		fmt.Fprintf(os.Stderr, "PilotDeck 服务未启动 (%s)，请先启动服务再重试。\n", cfg.PilotDeck.BaseURL)
		return 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted. Progress up to the last completed job has been saved.")
		cancel()
	}()

	start := time.Now()
	err = jobs.RunTask(ctx, &jobs.RunOptions{
		TaskDir: taskDir,
		Debug:   cfg.Debug,
		Cfg:     cfg,
		OnJobDone: func(jobName string, resp *pilotdeck.AgentResponse) {
			fmt.Printf("  ✓ %s (session: %s)\n", jobName, resp.SessionID)
		},
	})
	elapsed := time.Since(start)

	if err != nil {
		if ctx.Err() != nil {
			fmt.Fprintf(os.Stderr, "Stopped after %s.\n", elapsed.Round(time.Second))
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("\nTask completed in %s.\n", elapsed.Round(time.Second))
	return 0
}

// ===================== Config Mode =====================

func runConfigMode(taskName string) int {
	taskDir := task.TaskDir(taskName)
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		fmt.Printf("Task %q does not exist. Starting configuration wizard...\n", taskName)
		if err := task.CreateTaskWizard(taskName); err != nil {
			fmt.Fprintf(os.Stderr, "Wizard error: %v\n", err)
			return 1
		}
		return 0
	}

	cfg, err := task.LoadConfig(taskDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		return 1
	}
	if !checkServerHealth(cfg.PilotDeck.BaseURL) {
		fmt.Fprintf(os.Stderr, "PilotDeck 服务未启动 (%s)，配置修改仍可继续。\n", cfg.PilotDeck.BaseURL)
	}

	if err := task.ReconfigureWizard(taskName); err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		return 1
	}
	return 0
}

// ===================== Export Mode =====================

func runExportMode(taskName, format string) int {
	taskDir := task.TaskDir(taskName)
	cfg, err := task.LoadConfig(taskDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		return 1
	}
	if cfg.SessionID == "" {
		fmt.Println("该任务还没有运行过，无会话记录可导出。")
		return 0
	}

	fmt.Printf("正在获取会话 %s 的消息...\n", cfg.SessionID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rawMessages, err := pilotdeck.FetchSessionMessages(ctx,
		cfg.PilotDeck.BaseURL, cfg.PilotDeck.APIKey,
		cfg.SessionID, cfg.PilotDeck.ProjectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取消息失败: %v\n", err)
		return 1
	}

	exportDir := filepath.Join(taskDir, "export")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建导出目录失败: %v\n", err)
		return 1
	}

	ts := time.Now().Format("20060102_150405")
	var exportPath string
	var content []byte

	switch format {
	case "report":
		content = formatReport(rawMessages, cfg.SessionID)
		exportPath = filepath.Join(exportDir, fmt.Sprintf("report-%s-%s.md", cfg.SessionID, ts))
	case "detail":
		content = formatDetail(rawMessages, cfg.SessionID)
		exportPath = filepath.Join(exportDir, fmt.Sprintf("detail-%s-%s.md", cfg.SessionID, ts))
	default:
		content = rawMessages
		exportPath = filepath.Join(exportDir, fmt.Sprintf("session-%s-%s.json", cfg.SessionID, ts))
	}

	if err := os.WriteFile(exportPath, content, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入导出文件失败: %v\n", err)
		return 1
	}

	fmt.Printf("✓ 结果已导出到 %s\n", exportPath)
	return 0
}

// ===================== Formatters =====================


type sessionMessage struct {
	ID        string          `json:"id"`
	SessionID string          `json:"sessionId"`
	Timestamp string          `json:"timestamp"`
	Provider  string          `json:"provider"`
	Kind      string          `json:"kind"`
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolName  string          `json:"toolName,omitempty"`
	ToolInput json.RawMessage `json:"toolInput,omitempty"`
	ToolID    string          `json:"toolId,omitempty"`
	IsError   bool            `json:"isError,omitempty"`
	ErrorCode string          `json:"errorCode,omitempty"`
}

type sessionResponse struct {
	Messages []sessionMessage `json:"messages"`
	Total    int              `json:"total"`
	HasMore  bool             `json:"hasMore"`
}

func parseMessages(raw []byte) (*sessionResponse, error) {
	var resp sessionResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func formatReport(raw []byte, sessionID string) []byte {
	resp, err := parseMessages(raw)
	if err != nil {
		return raw
	}

	var b strings.Builder
	b.WriteString("# 任务会话报告\n\n")
	b.WriteString(fmt.Sprintf("**会话ID**: `%s`\n", sessionID))
	b.WriteString(fmt.Sprintf("**消息总数**: %d\n", resp.Total))
	b.WriteString(fmt.Sprintf("**导出时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	var userMsgs, assistantMsgs, toolCalls, toolResults, thinking []sessionMessage
	for _, m := range resp.Messages {
		switch m.Kind {
		case "text":
			if m.Role == "user" {
				userMsgs = append(userMsgs, m)
			} else {
				assistantMsgs = append(assistantMsgs, m)
			}
		case "thinking":
			thinking = append(thinking, m)
		case "tool_use":
			toolCalls = append(toolCalls, m)
		case "tool_result":
			toolResults = append(toolResults, m)
		}
	}

	b.WriteString("## 概览\n\n| 类型 | 数量 |\n|---|---|\n")
	b.WriteString(fmt.Sprintf("| 用户消息 | %d |\n", len(userMsgs)))
	b.WriteString(fmt.Sprintf("| 助手回复 | %d |\n", len(assistantMsgs)))
	b.WriteString(fmt.Sprintf("| 工具调用 | %d |\n", len(toolCalls)))
	b.WriteString(fmt.Sprintf("| 工具结果 | %d |\n", len(toolResults)))
	b.WriteString(fmt.Sprintf("| 推理过程 | %d |\n", len(thinking)))

	b.WriteString("\n## 用户请求\n\n")
	for _, m := range userMsgs {
		b.WriteString(fmt.Sprintf("- **Q**: %s\n", truncateMsg(m.Content, 200)))
	}

	b.WriteString("\n## 助手回复摘要\n\n")
	for _, m := range assistantMsgs {
		b.WriteString(fmt.Sprintf("- **A**: %s\n", truncateMsg(m.Content, 300)))
	}

	b.WriteString("\n## 工具使用\n\n")
	for _, m := range toolCalls {
		input := string(m.ToolInput)
		if len(input) > 100 {
			input = input[:100] + "..."
		}
		status := "✅"
		for _, r := range toolResults {
			if r.ToolID == m.ToolID {
				if r.IsError {
					status = "❌"
				}
				break
			}
		}
		b.WriteString(fmt.Sprintf("- %s `%s`: `%s`\n", status, m.ToolName, input))
	}

	var hasErrors bool
	for _, r := range toolResults {
		if r.IsError {
			hasErrors = true
			break
		}
	}
	if hasErrors {
		b.WriteString("\n## 错误信息\n\n")
		for _, r := range toolResults {
			if r.IsError {
				b.WriteString(fmt.Sprintf("- ❌ `%s`: %s\n", r.ToolID, truncateMsg(r.Content, 200)))
			}
		}
	}

	return []byte(b.String())
}

func formatDetail(raw []byte, sessionID string) []byte {
	resp, err := parseMessages(raw)
	if err != nil {
		return raw
	}

	var b strings.Builder
	b.WriteString("# 完整交互记录\n\n")
	b.WriteString(fmt.Sprintf("**会话ID**: `%s`\n", sessionID))
	b.WriteString(fmt.Sprintf("**导出时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString("---\n\n")

	for _, m := range resp.Messages {
		ts := m.Timestamp
		if len(ts) >= 19 {
			ts = ts[:19]
		}

		switch m.Kind {
		case "text":
			if m.Role == "user" {
				b.WriteString(fmt.Sprintf("### 🧑 用户 (%s)\n\n%s\n\n", ts, m.Content))
			} else {
				b.WriteString(fmt.Sprintf("### 🤖 助手 (%s)\n\n%s\n\n", ts, m.Content))
			}
		case "thinking":
			b.WriteString(fmt.Sprintf("> 💭 思考 (%s)\n> %s\n\n", ts, m.Content))
		case "tool_use":
			input := formatToolInput(m.ToolInput)
			b.WriteString(fmt.Sprintf("**🔧 工具调用: `%s`** (%s)\n\n", m.ToolName, ts))
			b.WriteString(fmt.Sprintf("```json\n%s\n```\n\n", input))
		case "tool_result":
			icon := "✅"
			if m.IsError {
				icon = "❌"
			}
			b.WriteString(fmt.Sprintf("**%s 工具结果** (%s)\n\n", icon, ts))
			b.WriteString(fmt.Sprintf("```\n%s\n```\n\n", truncateMsg(m.Content, 2000)))
		}
	}

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("\n*共 %d 条消息*\n", resp.Total))
	return []byte(b.String())
}

func truncateMsg(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func formatToolInput(input json.RawMessage) string {
	if len(input) == 0 {
		return "(empty)"
	}
	var pretty map[string]interface{}
	if err := json.Unmarshal(input, &pretty); err == nil {
		formatted, _ := json.MarshalIndent(pretty, "", "  ")
		return string(formatted)
	}
	return string(input)
}

// ===================== Status Mode =====================

func runStatusMode() int {
	tasks, err := task.ListTasks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if len(tasks) == 0 {
		fmt.Println("没有任务。")
		return 0
	}

	fmt.Println("\n--- 任务状态 ---")
	for _, t := range tasks {
		statusLine := taskStatusLine(t)
		fmt.Printf("  %s %s\n", t, statusLine)
	}
	return 0
}

func taskStatusLine(taskName string) string {
	taskDir := task.TaskDir(taskName)
	cfg, err := task.LoadConfig(taskDir)
	if err != nil {
		return "(配置错误: " + err.Error() + ")"
	}

	completed, _ := task.CompletedJobs(taskDir)
	jobs, _ := task.ListJobs(taskDir)
	parts := []string{fmt.Sprintf("%d/%d jobs", len(completed), len(jobs))}

	if cfg.SessionID != "" {
		parts = append(parts, fmt.Sprintf("session: %s", cfg.SessionID))
	}
	if cfg.Debug {
		parts = append(parts, "debug")
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func checkServerHealth(baseURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return pilotdeck.HealthCheck(ctx, baseURL) == nil
}
