package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
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

	log.Printf("print.it rodando em http://%s/printit/", addr)
	log.Printf("interface web: http://%s/ (redireciona para /printit/)", addr)
	log.Printf("impressora: %s", cfg.printerAddr())
	log.Printf("config: %s", configFilePath())

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
