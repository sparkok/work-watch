package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"work-watch/internal/i18n"
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
		fmt.Printf(i18n.T("health.check"), baseURL)
	} else {
		fmt.Fprintf(os.Stderr, i18n.T("health.unreachable"), baseURL)
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

	// Initialize i18n
	_ = i18n.Init(task.AppDir)

	// Initialize/refresh global config from PilotDeck settings
	if err := task.InitGlobalConfig(); err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("error.init_config"), err)
	}

	// Apply configured language, if any
	if gc, err := task.LoadGlobalConfig(); err == nil && gc.Lang != "" {
		_ = i18n.SetLang(gc.Lang)
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

	if subcommand == "reset" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, i18n.T("error.not_exist"))
			return 1
		}
		return runResetMode(args[1])
	}

	// Otherwise treat as task name (direct run)
	return runTaskMode(subcommand)
}

// ===================== Interactive Menu =====================

func runMenuMode() int {
	menuCtx, menuCancel := context.WithCancel(context.Background())
	defer menuCancel() // safety: cancel if menu exits by error/panic

	for {
		fmt.Println(i18n.T("menu.title"))
		fmt.Println(i18n.T("menu.option.config"))
		fmt.Println(i18n.T("menu.option.run"))
		fmt.Println(i18n.T("menu.option.export"))
		fmt.Println(i18n.T("menu.option.status"))
		fmt.Println(i18n.T("menu.option.reset"))
		fmt.Println(i18n.T("menu.option.lang"))
		fmt.Println(i18n.T("menu.option.exit"))
		fmt.Print(i18n.T("menu.prompt"))

		input := task.ReadLine()

		switch input {
		case "1", "配置":
			menuConfig()
		case "2", "执行":
			menuRun(menuCtx)
		case "3", "结果导出":
			menuExport()
		case "4", "状态":
			menuStatus()
		case "5", "重置":
			menuReset()
		case "6", "退出":
			menuCancel()
			time.Sleep(500 * time.Millisecond)
			fmt.Println(i18n.T("menu.option.goodbye"))
			return 0
		case "7", "语言":
			menuLanguage()
		default:
			fmt.Println(i18n.T("menu.option.invalid"))
		}
	}
}

func menuConfig() {
	tasks, _ := task.ListTasks()
	if len(tasks) > 0 {
		fmt.Println(i18n.T("config.select.title"))
		for i, t := range tasks {
			fmt.Printf("  %d. %s\n", i+1, t)
		}
		fmt.Printf(i18n.T("config.select.create_new"), len(tasks)+1)
		fmt.Print(i18n.T("config.select.prompt"))

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

	fmt.Print(i18n.T("config.select.create_prompt"))
	name := task.ReadLine()
	if name != "" {
		runTaskMode(name)
	}
}

func menuRun(menuCtx context.Context) {
	tasks, err := task.ListTasks()
	if err != nil || len(tasks) == 0 {
		fmt.Println(i18n.T("task.no_tasks"))
		return
	}

	fmt.Println(i18n.T("task.select.title"))
	for i, t := range tasks {
		statusLine := taskStatusLine(t)
		fmt.Printf("  %d. %s %s\n", i+1, t, statusLine)
	}

	fmt.Print(i18n.T("task.select.prompt"))
	taskName := chooseTask(tasks)
	if taskName == "" {
		fmt.Println(i18n.T("task.select.invalid"))
		return
	}

	// Check if task config exists — if not, enter wizard
	taskDir := task.TaskDir(taskName)
	if _, err := os.Stat(task.ConfigPath(taskDir)); os.IsNotExist(err) {
		fmt.Printf(i18n.T("task.wizard_needed"), taskName)
		if err := task.CreateTaskWizard(taskName); err != nil {
			fmt.Fprintf(os.Stderr, "Wizard error: %v\n", err)
			return
		}
		taskDir = task.TaskDir(taskName)

		// Refresh global config from PilotDeck since user is starting fresh
		if err := task.RefreshGlobalConfigFromPilotDeck(); err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("error.refresh"), err)
			fmt.Fprintln(os.Stderr, i18n.T("error.refresh_hint"))
		}
	}

	cfg, err := task.LoadConfig(taskDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("error.config_load"), err)
		return
	}
	if !checkServerHealth(cfg.PilotDeck.BaseURL) {
		fmt.Fprintf(os.Stderr, i18n.T("error.server_down"), cfg.PilotDeck.BaseURL)
		return
	}

	// Prevent re-running a task that already has an active session
	if err := checkTaskNotRunning(context.Background(), cfg, taskDir); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	go func() {
		fmt.Printf(i18n.T("task.starting"), taskName)

		ctx, cancel := context.WithTimeout(menuCtx, 300*time.Second)
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println(i18n.T("task.interrupted"))
			cancel()
		}()

		start := time.Now()
		err := jobs.RunTask(ctx, &jobs.RunOptions{
			TaskDir: taskDir,
			Debug:   cfg.Debug,
			Cfg:     cfg,
			OnJobDone: func(jobName string, resp *pilotdeck.AgentResponse) {
				fmt.Printf(i18n.T("task.job_done"), jobName, resp.SessionID)
			},
		})
		elapsed := time.Since(start)

		if err != nil {
			if ctx.Err() != nil {
				fmt.Printf(i18n.T("task.stopped"), taskName, elapsed.Round(time.Second))
			} else {
				fmt.Fprintf(os.Stderr, i18n.T("task.failed"), taskName, err)
			}
		} else {
			fmt.Printf(i18n.T("task.completed"), taskName, elapsed.Round(time.Second))
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
		fmt.Println(i18n.T("status.no_tasks_export"))
		return
	}

	fmt.Println(i18n.T("export.select.title"))
	for i, t := range tasks {
		taskDir := task.TaskDir(t)
		cfg, _ := task.LoadConfig(taskDir)
		sid := ""
		if cfg != nil && cfg.SessionID != "" {
			sid = " (session: " + cfg.SessionID + ")"
		}
		fmt.Printf("  %d. %s%s\n", i+1, t, sid)
	}

	fmt.Print(i18n.T("export.select.prompt"))
	taskName := chooseTask(tasks)
	if taskName == "" {
		fmt.Println(i18n.T("export.select.invalid"))
		return
	}

	fmt.Println(i18n.T("export.format.title"))
	fmt.Println(i18n.T("export.format.json"))
	fmt.Println(i18n.T("export.format.report"))
	fmt.Println(i18n.T("export.format.detail"))
	fmt.Print(i18n.T("export.format.prompt"))

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

func menuReset() {
	tasks, err := task.ListTasks()
	if err != nil || len(tasks) == 0 {
		fmt.Println(i18n.T("status.no_tasks_export"))
		return
	}

	fmt.Println(i18n.T("reset.select.title"))
	for i, t := range tasks {
		statusLine := taskStatusLine(t)
		fmt.Printf("  %d. %s %s\n", i+1, t, statusLine)
	}

	fmt.Print(i18n.T("reset.select.prompt"))
	taskName := chooseTask(tasks)
	if taskName == "" {
		fmt.Println(i18n.T("reset.select.invalid"))
		return
	}

	runResetMode(taskName)
}

func menuLanguage() {
	langs := i18n.Available()
	fmt.Println(i18n.T("lang.title"))

	codes := make([]string, 0, len(langs))
	for code, entry := range langs {
		codes = append(codes, code)
		fmt.Printf("  %d. %s\n", len(codes), entry.Label)
	}

	fmt.Printf(i18n.T("lang.select.prompt"), len(codes))
	input := task.ReadLine()
	n, err := strconv.Atoi(input)
	if err != nil || n < 1 || n > len(codes) {
		fmt.Println(i18n.T("lang.invalid"))
		return
	}

	code := codes[n-1]
	if err := i18n.SetLang(code); err != nil {
		fmt.Fprintf(os.Stderr, "Switch lang error: %v\n", err)
		return
	}

	// Also persist to config.yaml
	if gc, err := task.LoadGlobalConfig(); err == nil {
		gc.Lang = code
		_ = task.SaveGlobalConfig(gc)
	}

	fmt.Printf(i18n.T("lang.changed"), langs[code].Label)
}

// ===================== Task Execution =====================

func runTaskMode(taskName string) int {
	taskDir := task.TaskDir(taskName)

	// Check if task config exists — if not, enter wizard
	if _, err := os.Stat(task.ConfigPath(taskDir)); os.IsNotExist(err) {
		fmt.Printf(i18n.T("task.wizard_needed"), taskName)
		if err := task.CreateTaskWizard(taskName); err != nil {
			fmt.Fprintf(os.Stderr, "Wizard error: %v\n", err)
			return 1
		}
		taskDir = task.TaskDir(taskName)

		// Refresh global config from PilotDeck since the user is starting fresh
		if err := task.RefreshGlobalConfigFromPilotDeck(); err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("error.refresh"), err)
			fmt.Fprintln(os.Stderr, i18n.T("error.refresh_hint"))
		}
	}

	cfg, err := task.LoadConfig(taskDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("error.config_load_en"), err)
		return 1
	}

	if !checkServerHealth(cfg.PilotDeck.BaseURL) {
		fmt.Fprintf(os.Stderr, i18n.T("error.server_down_retry"), cfg.PilotDeck.BaseURL)
		return 0
	}

	// Prevent re-running a task that already has an active session
	if err := checkTaskNotRunning(context.Background(), cfg, taskDir); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println(i18n.T("task.interrupted"))
		cancel()
	}()

	start := time.Now()
	err = jobs.RunTask(ctx, &jobs.RunOptions{
		TaskDir: taskDir,
		Debug:   cfg.Debug,
		Cfg:     cfg,
		OnJobDone: func(jobName string, resp *pilotdeck.AgentResponse) {
			fmt.Printf(i18n.T("task.job_done"), jobName, resp.SessionID)
		},
	})
	elapsed := time.Since(start)

	if errors.Is(err, jobs.ErrConnectionFailed) {
		fmt.Fprintf(os.Stderr, "PilotDeck 连接失败（已重试，请检查服务状态后重新运行）。\n")
		return 0
	}
	if err != nil {
		if ctx.Err() != nil {
			fmt.Fprintf(os.Stderr, i18n.T("run.stopped_short"), elapsed.Round(time.Second))
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf(i18n.T("run.task_completed"), elapsed.Round(time.Second))

	// Check if already confirmed in a previous run
	alreadyConfirmed, checkErr := task.IsConfirmed(taskDir)
	if checkErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to check confirmation status: %v\n", checkErr)
	}
	if alreadyConfirmed {
		fmt.Println(i18n.T("confirm.already"))
		return 0
	}

	// ===== Post-task confirmation with PilotDeck =====
	fmt.Println(i18n.T("confirm.starting"))

	// Reload config to get the session ID saved by runner
	cfg, err = task.LoadConfig(taskDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to reload config: %v\n", err)
	}

	confirmed, confirmErr := confirmTaskOutcome(context.Background(), cfg)
	if confirmErr != nil {
		fmt.Fprintf(os.Stderr, i18n.T("confirm.failure"), confirmErr)
		return 1
	}
	if confirmed {
		_ = task.MarkConfirmed(taskDir)
		fmt.Println(i18n.T("confirm.success"))
		return 0
	}
	fmt.Fprintln(os.Stderr, i18n.T("confirm.fail_outcome"))
	return 1
}

// confirmTaskOutcome waits for the PilotDeck session to finish processing via WebSocket,
// then fetches messages to confirm success. Fails if no messages exist.
// Returns true (success), false (failure), or error.
func confirmTaskOutcome(ctx context.Context, cfg *task.TaskConfig) (bool, error) {
	if cfg == nil || cfg.PilotDeck.BaseURL == "" {
		return false, fmt.Errorf("PilotDeck 配置不完整")
	}
	sessionID := cfg.SessionID
	if sessionID == "" {
		return false, fmt.Errorf("没有任务会话记录，无法确认")
	}

	// 等待 isProcessing 变为 false
	status, err := pilotdeck.CheckSessionStatus(ctx, cfg.PilotDeck.BaseURL, sessionID)
	if err != nil {
		return false, fmt.Errorf("检查会话状态失败: %w", err)
	}
	if status == nil {
		return false, fmt.Errorf("会话状态未知")
	}

	// isProcessing=false 后，确认有消息即成功
	messagesJSON, err := pilotdeck.FetchSessionMessages(ctx,
		cfg.PilotDeck.BaseURL,
		cfg.PilotDeck.APIKey,
		sessionID,
		cfg.PilotDeck.ProjectPath,
	)
	if err != nil {
		return false, fmt.Errorf("获取会话消息失败: %w", err)
	}
	if len(messagesJSON) == 0 || string(messagesJSON) == `{"messages":null,"total":0}` {
		return false, fmt.Errorf("会话无消息，按失败处理")
	}

	return true, nil
}

// checkTaskNotRunning prevents re-running a task that is already executing.
// The .running marker file is the authoritative signal; a stale marker (>1h)
// is treated as crash recovery and removed.
//
// In discrete mode, each run is independent — always remove any stale .running
// and proceed. In continuous mode, block re-run if another process holds the marker.
// Returns nil if safe to run, or an error describing why not.
func checkTaskNotRunning(ctx context.Context, cfg *task.TaskConfig, taskDir string) error {
	// Discrete mode: always allow re-run — if .running exists, remove it and re-create.
	if cfg.Mode == "discrete" {
		_ = task.RemoveRunningMarker(taskDir)
		_, err := task.TryAcquireRunningMarker(taskDir, "")
		return err // nil if acquired, error only on I/O failure
	}

	// Continuous mode: atomically acquire; block if another process holds it.
	acquired, err := task.TryAcquireRunningMarker(taskDir, cfg.SessionID)
	if err != nil {
		return fmt.Errorf("检查运行状态失败: %w", err)
	}
	if acquired {
		return nil
	}

	// Someone else holds the marker — read it for staleness check
	marker, err := task.ReadRunningMarker(taskDir)
	if err != nil {
		return fmt.Errorf("检查运行状态失败: %w", err)
	}
	if marker == nil {
		// Vanished between acquire-fail and read — race, but safe to proceed
		return nil
	}

	// Staleness check: crash recovery after 1 hour
	started, err := time.Parse(time.RFC3339, marker.Started)
	if err == nil && time.Since(started) > 1*time.Hour {
		fmt.Fprintln(os.Stderr, "Warning: .running marker is stale (>1h), removing it.")
		_ = task.RemoveRunningMarker(taskDir)
		acquired, err := task.TryAcquireRunningMarker(taskDir, cfg.SessionID)
		if err != nil {
			return fmt.Errorf("检查运行状态失败: %w", err)
		}
		if acquired {
			return nil
		}
		// Couldn't re-acquire — someone else grabbed it, block
	}

	sid := marker.SessionID
	if sid == "" {
		sid = cfg.SessionID
	}
	if sid == "" {
		return fmt.Errorf("任务正在执行中，请等待完成后再试")
	}
	return fmt.Errorf("任务正在执行中 (session: %s)，请等待完成后再试", sid)
}

// ===================== Config Mode =====================

func runConfigMode(taskName string) int {
	taskDir := task.TaskDir(taskName)
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		fmt.Printf(i18n.T("error.not_exist_config"), taskName)
		if err := task.CreateTaskWizard(taskName); err != nil {
			fmt.Fprintf(os.Stderr, "Wizard error: %v\n", err)
			return 1
		}
		return 0
	}

	cfg, err := task.LoadConfig(taskDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("error.config_load_en"), err)
		return 1
	}
	if !checkServerHealth(cfg.PilotDeck.BaseURL) {
		fmt.Fprintf(os.Stderr, i18n.T("error.server_down_config"), cfg.PilotDeck.BaseURL)
	}

	if err := task.ReconfigureWizard(taskName); err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("error.config_load_en"), err)
		return 1
	}
	return 0
}

// ===================== Reset Mode =====================

func runResetMode(taskName string) int {
	taskDir := task.TaskDir(taskName)
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, i18n.T("error.not_exist"), taskName)
		return 1
	}

	fmt.Printf(i18n.T("reset.prompt"), taskName)
	fmt.Print(i18n.T("reset.confirm"))
	confirm := task.ReadLine()
	if confirm != "yes" {
		fmt.Println(i18n.T("reset.cancelled"))
		return 0
	}

	if err := task.ResetTask(taskDir); err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("error.reset"), err)
		return 1
	}

	fmt.Printf(i18n.T("reset.success"), taskName)
	return 0
}

func runExportMode(taskName, format string) int {
	taskDir := task.TaskDir(taskName)
	cfg, err := task.LoadConfig(taskDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("error.config_load_en"), err)
		return 1
	}
	if cfg.SessionID == "" {
		fmt.Println(i18n.T("export.no_session"))
		return 0
	}

	fmt.Printf(i18n.T("export.fetching"), cfg.SessionID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rawMessages, err := pilotdeck.FetchSessionMessages(ctx,
		cfg.PilotDeck.BaseURL, cfg.PilotDeck.APIKey,
		cfg.SessionID, cfg.PilotDeck.ProjectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("error.fetch"), err)
		return 1
	}

	exportDir := filepath.Join(taskDir, "export")
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("error.export_dir"), err)
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

	if err := os.WriteFile(exportPath, content, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("error.export_file"), err)
		return 1
	}

	fmt.Printf(i18n.T("export.success"), exportPath)
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
		fmt.Println(i18n.T("status.no_tasks"))
		return 0
	}

	fmt.Println(i18n.T("status.title"))
	for _, t := range tasks {
		statusLine := taskStatusLine(t)
		fmt.Printf("  %s %s\n", t, statusLine)
	}
	return 0
}

func taskStatusLine(taskName string) string {
	taskDir := task.TaskDir(taskName)

	// Check if task config file exists
	if _, err := os.Stat(task.ConfigPath(taskDir)); os.IsNotExist(err) {
		return i18n.T("status.unconfigured")
	}

	cfg, err := task.LoadConfig(taskDir)
	if err != nil {
		return fmt.Sprintf(i18n.T("status.config_error"), err.Error())
	}

	completed, _ := task.CompletedJobs(taskDir)
	jobs, _ := task.ListJobs(taskDir)
	parts := []string{fmt.Sprintf("%d/%d jobs", len(completed), len(jobs))}

	if cfg.SessionID != "" {
		parts = append(parts, fmt.Sprintf("session: %s", cfg.SessionID))
		if len(completed) < len(jobs) {
			parts = append(parts, i18n.T("status.running"))
		}
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
