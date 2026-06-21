package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

const singletonPort = 9289

func setupRuntimeLogging() error {
	if err := ensureDir(logsDir()); err != nil {
		return err
	}

	logPath := logFilePath()
	if info, err := os.Stat(logPath); err == nil && info.Size() > 5<<20 {
		_ = os.Rename(logPath, logPath+".1")
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	log.SetOutput(io.MultiWriter(os.Stdout, file))
	log.SetFlags(log.LstdFlags)
	return nil
}

func acquireSingleInstance() (func(), bool) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", singletonPort))
	if err != nil {
		return nil, true
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
