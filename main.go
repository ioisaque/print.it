package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	if handleCLI() {
		return
	}

	release, alreadyRunning := acquireSingleInstance()
	if alreadyRunning {
		log.Println("print.it ja esta em execucao")
		return
	}
	defer release()

	if err := setupRuntimeLogging(); err != nil {
		log.Printf("aviso: log em arquivo indisponivel: %v", err)
	}

	if err := loadConfig(); err != nil {
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
	if dir := webDevDir(); dir != "" {
		log.Printf("ui dev: servindo arquivos de %s/ (sem rebuild)", dir)
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
