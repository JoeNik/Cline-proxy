package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Config struct {
	AdminPassword       string
	AutoRefreshCron     string
	AutoRefreshEnabled  bool
}

var (
	globalConfig   *Config
	configMu       sync.RWMutex
	configPath     string
)

func init() {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)

	// Try .env in executable directory first
	configPath = filepath.Join(dir, ".env")
	if !fileExists(configPath) {
		// Try current working directory
		if cwd, err := os.Getwd(); err == nil {
			configPath = filepath.Join(cwd, ".env")
		}
	}
}

func loadConfig() *Config {
	configMu.RLock()
	if globalConfig != nil {
		defer configMu.RUnlock()
		return globalConfig
	}
	configMu.RUnlock()

	configMu.Lock()
	defer configMu.Unlock()

	cfg := &Config{
		AdminPassword:      "",
		AutoRefreshCron:    "0 */6 * * *",
		AutoRefreshEnabled: true,
	}

	if !fileExists(configPath) {
		globalConfig = cfg
		return cfg
	}

	file, err := os.Open(configPath)
	if err != nil {
		globalConfig = cfg
		return cfg
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "ADMIN_PASSWORD":
			cfg.AdminPassword = value
		case "AUTO_REFRESH_CRON":
			cfg.AutoRefreshCron = value
		case "AUTO_REFRESH_ENABLED":
			cfg.AutoRefreshEnabled = (value == "true" || value == "1")
		}
	}

	globalConfig = cfg
	return cfg
}

func saveConfig(cfg *Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	content := "# Admin panel password (leave empty to disable password protection)\n"
	content += "ADMIN_PASSWORD=" + cfg.AdminPassword + "\n\n"
	content += "# Auto refresh cron expression (e.g., \"0 */6 * * *\" = every 6 hours)\n"
	content += "# Format: minute hour day month weekday\n"
	content += "AUTO_REFRESH_CRON=" + cfg.AutoRefreshCron + "\n\n"
	content += "# Enable auto refresh (true/false)\n"
	if cfg.AutoRefreshEnabled {
		content += "AUTO_REFRESH_ENABLED=true\n"
	} else {
		content += "AUTO_REFRESH_ENABLED=false\n"
	}

	err := os.WriteFile(configPath, []byte(content), 0600)
	if err == nil {
		globalConfig = cfg
	}
	return err
}

func reloadConfig() {
	configMu.Lock()
	globalConfig = nil
	configMu.Unlock()
	loadConfig()
}
