package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const singletonPort = 9289

func appendStartupLog(msg string) {
	dir := logsDir()
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "startup.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), msg)
}

func setupRuntimeLogging() error {
	if err := ensureDir(logsDir()); err != nil {
		appendStartupLog("mkdir logs: " + err.Error())
		return err
	}

	logPath := logFilePath()
	if info, err := os.Stat(logPath); err == nil && info.Size() > 5<<20 {
		_ = os.Rename(logPath, logPath+".1")
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		appendStartupLog("open log: " + err.Error())
		return err
	}

	log.SetOutput(io.MultiWriter(os.Stdout, file))
	log.SetFlags(log.LstdFlags)
	return nil
}

func agentAlreadyHealthy() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:9280/printit/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func acquireSingleInstance() (func(), bool) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", singletonPort))
	if err != nil {
		appendStartupLog(fmt.Sprintf("singleton ocupado (%v)", err))
		if agentAlreadyHealthy() {
			appendStartupLog("instancia saudavel ja ativa, saindo")
			return nil, true
		}
		appendStartupLog("sem instancia saudavel; tentando iniciar mesmo assim")
		return func() {}, false
	}

	if err := ensureDir(dataDir()); err != nil {
		log.Printf("aviso: nao foi possivel criar pasta de dados: %v", err)
	}

	_ = os.WriteFile(lockFilePath(), []byte(strconv.Itoa(os.Getpid())), 0o644)

	return func() {
		_ = listener.Close()
		_ = os.Remove(lockFilePath())
	}, false
}

func handleCLI() bool {
	if len(os.Args) < 2 {
		return false
	}

	switch os.Args[1] {
	case "--version", "-version", "-v", "version":
		fmt.Println(version)
		return true
	case "--uninstall", "uninstall":
		runUninstall()
		return true
	case "--logs", "logs":
		showLogPaths()
		return true
	}

	return false
}

func runUninstall() {
	var script string
	switch runtime.GOOS {
	case "darwin":
		script = "/usr/local/share/print.it/uninstall.sh"
		if _, err := os.Stat(script); err != nil {
			script = filepath.Join("packaging", "macos", "uninstall.sh")
		}
	case "windows":
		script = filepath.Join(os.Getenv("ProgramFiles"), "print.it", "uninstall.ps1")
		if _, err := os.Stat(script); err != nil {
			script = filepath.Join("packaging", "windows", "uninstall.ps1")
		}
	default:
		script = "/usr/share/print.it/uninstall.sh"
		if _, err := os.Stat(script); err != nil {
			script = filepath.Join("packaging", "linux", "uninstall.sh")
		}
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-File", script)
	default:
		cmd = exec.Command("/bin/bash", script)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("desinstalacao falhou: %v", err)
	}
}

func showLogPaths() {
	fmt.Println("Logs:")
	fmt.Printf("  %s\n", logFilePath())
	fmt.Printf("  %s\n", filepath.Join(logsDir(), "startup.log"))
	if runtime.GOOS == "windows" {
		fmt.Printf("  %s\n", filepath.Join(os.Getenv("ProgramData"), "print.it", "install.log"))
	}
	fmt.Printf("Config: %s\n", configFilePath())
}
