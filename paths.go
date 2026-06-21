package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func dataDir() string {
	if env := os.Getenv("PRINT_IT_DATA"); env != "" {
		return env
	}

	dir, err := os.UserConfigDir()
	if err == nil && dir != "" {
		return filepath.Join(dir, "print.it")
	}

	return legacyBinDir()
}

func logsDir() string {
	if runtime.GOOS == "windows" {
		if dir := os.Getenv("ProgramData"); dir != "" {
			return filepath.Join(dir, "print.it", "logs")
		}
	}
	return filepath.Join(dataDir(), "logs")
}

func logFilePath() string {
	return filepath.Join(logsDir(), "print.it.log")
}

func lockFilePath() string {
	return filepath.Join(dataDir(), "print.it.lock")
}

func legacyBinDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}

	dir := filepath.Dir(exe)
	if dir == "" || dir == "." {
		wd, _ := os.Getwd()
		return wd
	}
	return dir
}

func configFilePath() string {
	if env := os.Getenv("PRINT_IT_CONFIG"); env != "" {
		return env
	}

	if _, err := os.Stat("config.json"); err == nil {
		return "config.json"
	}

	legacy := filepath.Join(legacyBinDir(), "config.json")
	if _, err := os.Stat(legacy); err == nil {
		if strings.Contains(legacyBinDir(), "go-build") || strings.Contains(legacyBinDir(), string(os.PathSeparator)+"tmp"+string(os.PathSeparator)) {
			return legacy
		}
		if _, err := os.Stat(filepath.Join(dataDir(), "config.json")); os.IsNotExist(err) {
			return legacy
		}
	}

	return filepath.Join(dataDir(), "config.json")
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
