package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	appendStartupLog("inicio pid=" + strconv.Itoa(os.Getpid()))

	if handleCLI() {
		return
	}

	if err := setupRuntimeLogging(); err != nil {
		appendStartupLog("log em arquivo indisponivel: " + err.Error())
	}

	release, alreadyRunning := acquireSingleInstance()
	if alreadyRunning {
		log.Println("print.it ja esta em execucao")
		appendStartupLog("saindo: ja em execucao")
		return
	}
	defer release()

	if err := loadConfig(); err != nil {
		appendStartupLog("config: " + err.Error())
		log.Fatalf("config: %v", err)
	}

	cfg := getConfig()
	addr := cfg.listenAddr()

	server := &http.Server{
		Addr:              addr,
		Handler:           newRouter(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
	}

	log.Printf("print.it %s rodando em http://%s/printit/", version, addr)
	log.Printf("interface web: http://%s/ (redireciona para /printit/)", addr)
	log.Printf("impressora: %s", cfg.printerAddr())
	log.Printf("config: %s", configFilePath())
	log.Printf("dados: %s", dataDir())
	log.Printf("logs: %s", logFilePath())
	if dir := webDevDir(); dir != "" {
		log.Printf("ui dev: servindo arquivos de %s/ (sem rebuild)", dir)
	}

	appendStartupLog("escutando em " + addr)
	if err := server.ListenAndServe(); err != nil {
		appendStartupLog("listen: " + err.Error())
		log.Fatal(err)
	}
}
