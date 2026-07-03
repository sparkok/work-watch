package i18n

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LangEntry is one entry in lang.yml.
type LangEntry struct {
	Label string `yaml:"label"`
	File  string `yaml:"file"`
}

type translations map[string]string

var (
	current     translations
	fallback    translations
	available   map[string]LangEntry
	currentLang string
)

// builtinZH holds the canonical Chinese translations compiled into the binary.
// These are the source of truth and always available even if zh.yml is missing.
// When zh.yml exists on disk, it overrides the built-ins.
var builtinZH = translations{
	"menu.title":          "========== Work-Watch 任务监工 ==========",
	"menu.option.config":  " 1. 配置    — 创建或修改任务配置",
	"menu.option.run":     " 2. 执行    — 执行任务 (异步)",
	"menu.option.export":  " 3. 结果导出 — 导出任务会话记录",
	"menu.option.status":  " 4. 状态    — 查看任务状态",
	"menu.option.reset":   " 5. 重置    — 将任务恢复为初始未执行状态",
	"menu.option.lang":    " 7. 语言    — 切换显示语言",
	"menu.option.exit":    " 6. 退出",
	"menu.prompt":         "\n请选择 (1-7): ",
	"menu.option.invalid": "无效选择，请重新输入。",
	"menu.option.goodbye": "再见!",

	"config.select.title":         "\n--- 已有任务 ---",
	"config.select.create_new":    "  %d. 创建新任务",
	"config.select.prompt":        "\n选择任务进行配置, 或创建新任务: ",
	"config.select.create_prompt": "\n输入新任务名称: ",

	"task.select.title":   "\n--- 选择要执行的任务 ---",
	"task.select.prompt":  "\n选择任务: ",
	"task.select.invalid": "无效选择。",
	"task.no_tasks":       "没有可用的任务。请先创建任务。",
	"task.starting":       "\n▶ 开始异步执行任务: %s\n",
	"task.interrupted":    "\n任务被中断。已完成进度已保存。",
	"task.job_done":       "  ✓ %s (session: %s)\n",
	"task.stopped":        "⏹ 任务 %s 已停止 (%s)\n",
	"task.failed":         "✗ 任务 %s 失败: %v\n",
	"task.completed":      "✓ 任务 %s 完成 (%s)\n",
	"task.wizard_needed":  "Task %q 需要配置。正在启动配置向导…\n",

	"export.select.title":   "\n--- 选择要导出结果的任务 ---",
	"export.select.prompt":  "\n选择任务: ",
	"export.select.invalid": "无效选择。",
	"export.format.title":   "\n选择导出格式:",
	"export.format.json":    "  1. JSON       — 机器可读，可用于分析",
	"export.format.report":  "  2. 报告       — 简明报告，辅助决策",
	"export.format.detail":  "  3. 详细交互   — 完整交互过程，方便阅读",
	"export.format.prompt":  "\n请选择 (1-3): ",
	"export.no_session":     "该任务还没有运行过，无会话记录可导出。",
	"export.fetching":       "正在获取会话 %s 的消息…\n",
	"export.success":        "✓ 结果已导出到 %s\n",

	"reset.select.title":   "\n--- 选择要重置的任务 ---",
	"reset.select.prompt":  "\n选择任务: ",
	"reset.select.invalid": "无效选择。",
	"reset.prompt":         "确定要重置任务 %q 吗？这将清除执行状态、日志和会话信息，但保留 job 定义文件。\n",
	"reset.confirm":        "输入 yes 确认重置: ",
	"reset.cancelled":      "已取消重置。",
	"reset.success":        "✓ 任务 %q 已重置为初始状态。\n",

	"status.title":           "\n--- 任务状态 ---",
	"status.no_tasks":        "没有任务。",
	"status.unconfigured":    "(未配置)",
	"status.config_error":    "(配置错误: %s)",
	"status.running":         "执行中",
	"status.no_tasks_export": "没有可用的任务。",

	"lang.title":         "\n--- 选择语言 / Select Language ---",
	"lang.select.prompt": "\n选择 (1-%d): ",
	"lang.invalid":       "无效选择。",
	"lang.changed":       "语言已切换为 %s。",

	"confirm.starting":     "正在向 PilotDeck 确认任务结果…",
	"confirm.failure":      "确认失败: %v\n",
	"confirm.success":      "✅ PilotDeck 确认: 任务成功完成。",
	"confirm.fail_outcome": "❌ PilotDeck 确认: 任务失败。",
	"confirm.already":      "任务已被确认过，跳过确认步骤。",

	"run.no_tasks":        "没有可用的任务。",
	"run.task_completed":  "\nTask completed in %s.\n",
	"run.stopped_short":   "Stopped after %s.\n",
	"run.jobs_completed":  "All %d job(s) already completed. To re-run, delete the status file.\n",
	"run.no_job_files":    "No job files found in jobs/ directory.\n",
	"run.running":         "Running job: %s\n",
	"run.retry":           "  PilotDeck 连接失败（第 %d 次/共 %d 次）: %v\n",
	"run.warning_session": "Warning: failed to save session: %v\n",

	"runner.discrete_completed": "任务 %s: PilotDeck 会话已结束，当前作业已标记完成。\n",
	"runner.session_gone":       "任务 %s: PilotDeck 会话不存在，已重置保护锁。\n",
	"runner.session_done":       "任务 %s: PilotDeck 会话已完成，自动确认。\n",
	"runner.reconnect":          "任务正在 PilotDeck 中执行，重新连接会话 %s…\n",

	"health.check":       "✓ PilotDeck %s is healthy\n",
	"health.unreachable": "✗ PilotDeck %s is unreachable\n",

	"wizard.prompt_title":       "No global config.yaml found. Let's set up PilotDeck connection.",
	"wizard.prompt_project":     "Project path [%s]: ",
	"wizard.prompt_apikey":      "API key (optional): ",
	"wizard.prompt_apikey_note": "  (no API key will be sent)",
	"wizard.prompt_baseurl":     "Base URL [%s]: ",
	"wizard.no_api_key":         "(no API key will be sent)",

	"error.not_exist":          "任务 %q 不存在。\n",
	"error.not_exist_config":   "Task %q does not exist. Starting configuration wizard...\n",
	"error.config_load":        "配置错误: %v\n",
	"error.config_load_en":     "Config error: %v\n",
	"error.server_down":        "PilotDeck 服务未启动 (%s)\n",
	"error.server_down_retry":  "PilotDeck 服务未启动 (%s)，请先启动服务再重试。\n",
	"error.server_down_config": "PilotDeck 服务未启动 (%s)，配置修改仍可继续。\n",
	"error.running":            "任务正在执行中，请等待完成后再试",
	"error.running_session":    "任务正在执行中 (session: %s)，请等待完成后再试",
	"error.completed_block":    "任务已在 PilotDeck 中完成，已自动确认。",
	"error.refresh":            "Warning: 无法从 PilotDeck 刷新配置: %v\n",
	"error.refresh_hint":       "请检查 PilotDeck 配置文件 (~/.pilotdeck/pilotdeck.yaml) 是否存在。",
	"error.reset":              "重置失败: %v\n",
	"error.init_config":        "自动初始化配置失败: %v\n",
	"error.fetch":              "获取消息失败: %v\n",
	"error.export_dir":         "创建导出目录失败: %v\n",
	"error.export_file":        "写入导出文件失败: %v\n",
	"error.no_session":         "没有任务会话记录，无法确认",
}

type langRegistry struct {
	Languages map[string]LangEntry `yaml:"languages"`
}

func deepCopyTranslations(src translations) translations {
	dst := make(translations, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// Init reads the lang directory and populates available languages.
// If lang.yml is missing, it creates a default registry with zh and en.
// Built-in zh translations are always loaded as fallback.
func Init(appDir string) error {
	langDir := filepath.Join(appDir, "lang")
	registryPath := filepath.Join(langDir, "lang.yml")

	// Initialize fallback from built-in zh
	fallback = deepCopyTranslations(builtinZH)

	// Read the registry
	registry := langRegistry{}
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default lang.yml with zh and en
			registry = langRegistry{
				Languages: map[string]LangEntry{
					"zh": {Label: "中文", File: "zh.yml"},
					"en": {Label: "English", File: "en.yml"},
				},
			}
			if err := os.MkdirAll(langDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "i18n: failed to create lang dir: %v\n", err)
			} else {
				data, _ := yaml.Marshal(&registry)
				_ = os.WriteFile(registryPath, data, 0o644)
			}
		} else {
			fmt.Fprintf(os.Stderr, "i18n: failed to read lang.yml: %v\n", err)
		}
	} else {
		if err := yaml.Unmarshal(raw, &registry); err != nil {
			fmt.Fprintf(os.Stderr, "i18n: failed to parse lang.yml: %v\n", err)
			registry = langRegistry{
				Languages: map[string]LangEntry{
					"zh": {Label: "中文", File: "zh.yml"},
				},
			}
		}
	}

	if registry.Languages == nil {
		registry.Languages = map[string]LangEntry{
			"zh": {Label: "中文", File: "zh.yml"},
		}
	}

	available = registry.Languages
	i18nAppDir = appDir
	currentLang = "zh"

	// Load zh translations (disk overrides built-in)
	loadTranslations("zh")

	return nil
}

// SetLang switches the active language. Looks up code in available,
// reads the translation file, and sets current. On error, current
// language stays unchanged.
func SetLang(code string) error {
	if _, ok := available[code]; !ok {
		return fmt.Errorf("language %q not available", code)
	}

	loadTranslations(code)
	if current == nil {
		return fmt.Errorf("translation file for %q not found or unparseable", code)
	}

	currentLang = code
	return nil
}

// T looks up a key in the current translations, then falls back to zh,
// then returns the key itself if not found in either.
func T(key string, args ...any) string {
	tpl, ok := current[key]
	if !ok {
		tpl, ok = fallback[key]
		if !ok {
			return key
		}
	}
	if len(args) == 0 {
		return tpl
	}
	return fmt.Sprintf(tpl, args...)
}

// Available returns a copy of the registered languages map.
func Available() map[string]LangEntry {
	result := make(map[string]LangEntry, len(available))
	for k, v := range available {
		result[k] = v
	}
	return result
}

// CurrentLang returns the current language code.
func CurrentLang() string {
	return currentLang
}

// CurrentLabel returns the label of the current language, or the code.
func CurrentLabel() string {
	if entry, ok := available[currentLang]; ok {
		return entry.Label
	}
	return currentLang
}

// loadTranslations reads a translation file from disk and merges it
// into the built-in zh map (only for "zh"). For other languages, it
// sets current from the disk file only.
func loadTranslations(code string) {
	entry, ok := available[code]
	if !ok {
		return
	}

	t := loadTranslationsFile(code, entry.File)
	if t == nil {
		if code == "zh" {
			current = deepCopyTranslations(builtinZH)
		} else {
			current = nil
		}
		return
	}

	if code == "zh" {
		// Merge built-in zh with disk overrides
		merged := deepCopyTranslations(builtinZH)
		for k, v := range t {
			merged[k] = v
		}
		current = merged
	} else {
		current = t
	}
}

// loadTranslationsFile reads a YAML translation file from disk.
// Returns nil if the file doesn't exist or can't be parsed.
func loadTranslationsFile(code, filename string) translations {
	if i18nAppDir == "" {
		return nil
	}
	path := filepath.Join(i18nAppDir, "lang", filename)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var t translations
	if err := yaml.Unmarshal(raw, &t); err != nil {
		return nil
	}
	return t
}

// i18nAppDir is set by Init and used by loadTranslationsFile.
var i18nAppDir string
