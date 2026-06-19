package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxUploadBytes = 10 << 20 // 10 MB

type printTextRequest struct {
	Text              string `json:"text"`
	Cut               *bool  `json:"cut"`
	Align             string `json:"align"`
	Bold              bool   `json:"bold"`
	TrimTrailingBlank *bool  `json:"trim_trailing_blank"`
}

type printRawRequest struct {
	DataBase64 string `json:"data_base64"`
}

type printPDFRequest struct {
	PDFBase64         string `json:"pdf_base64"`
	Cut               *bool  `json:"cut"`
	CutBetweenPages   *bool  `json:"cut_between_pages"`
	TrimTrailingBlank *bool  `json:"trim_trailing_blank"`
}

type printImageRequest struct {
	ImageBase64       string `json:"image_base64"`
	Cut               *bool  `json:"cut"`
	TrimTrailingBlank *bool  `json:"trim_trailing_blank"`
}

type printBarcodeRequest struct {
	Type  string `json:"type"`
	Data  string `json:"data"`
	Label string `json:"label"`
	Align string `json:"align"`
	Cut   *bool  `json:"cut"`
}

type printQRCodeRequest struct {
	Data  string `json:"data"`
	Label string `json:"label"`
	Align string `json:"align"`
	Cut   *bool  `json:"cut"`
}

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("config: %v", err)
	}

	cfg := getConfig()

	mux := http.NewServeMux()
	mux.Handle("GET /{$}", uiHandler())
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /config", handleGetConfig)
	mux.HandleFunc("PUT /config", handlePutConfig)
	mux.HandleFunc("GET /discover", handleDiscover)
	mux.HandleFunc("POST /discover/apply", handleDiscoverApply)
	mux.HandleFunc("POST /print", handlePrintText)
	mux.HandleFunc("POST /print/raw", handlePrintRaw)
	mux.HandleFunc("POST /print/pdf", handlePrintPDF)
	mux.HandleFunc("POST /print/image", handlePrintImage)
	mux.HandleFunc("POST /print/test", handlePrintTest)
	mux.HandleFunc("POST /print/barcode", handlePrintBarcode)
	mux.HandleFunc("POST /print/qrcode", handlePrintQRCode)

	addr := cfg.listenAddr()
	server := &http.Server{
		Addr:              addr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
	}

	log.Printf("print.it rodando em http://%s", addr)
	log.Printf("interface web: http://%s", addr)
	log.Printf("impressora: %s", cfg.printerAddr())
	log.Printf("config: %s", configFilePath())

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		cfg := getConfig()

		allowOrigin := ""
		for _, allowed := range cfg.CorsOrigins {
			if allowed == "*" {
				allowOrigin = "*"
				break
			}
			if origin != "" && allowed == origin {
				allowOrigin = origin
				break
			}
		}

		if allowOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func trimTrailingBlankValue(form string, value *bool, fallback bool) bool {
	if form != "" {
		return form == "true" || form == "1"
	}
	return boolValue(value, fallback)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	cfg := getConfig()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"service":  "print.it",
		"version":  "0.1.0",
		"printer":  cfg.printerAddr(),
		"listen":   cfg.listenAddr(),
		"paper_mm": cfg.PaperWidthMM,
		"printable_mm": cfg.printableWidthMM(),
	})
}

func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, getConfig())
}

func handlePutConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "corpo invalido")
		return
	}

	var patch Config
	if err := json.Unmarshal(body, &patch); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}

	var flags struct {
		TrimTrailingBlank *bool `json:"trim_trailing_blank"`
	}
	_ = json.Unmarshal(body, &flags)

	cfg := getConfig()
	if patch.PrinterHost != "" {
		cfg.PrinterHost = patch.PrinterHost
	}
	if patch.PrinterPort > 0 {
		cfg.PrinterPort = patch.PrinterPort
	}
	if patch.ListenHost != "" {
		cfg.ListenHost = patch.ListenHost
	}
	if patch.ListenPort > 0 {
		cfg.ListenPort = patch.ListenPort
	}
	if patch.PaperWidthMM > 0 {
		cfg.PaperWidthMM = patch.PaperWidthMM
	}
	if patch.PrintableWidthMM > 0 {
		cfg.PrintableWidthMM = patch.PrintableWidthMM
	}
	if flags.TrimTrailingBlank != nil {
		cfg.TrimTrailingBlank = *flags.TrimTrailingBlank
	}
	if len(patch.CorsOrigins) > 0 {
		cfg.CorsOrigins = patch.CorsOrigins
	}

	updated, err := updateConfig(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func handlePrintText(w http.ResponseWriter, r *http.Request) {
	var req printTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		writeError(w, http.StatusBadRequest, "campo text obrigatorio")
		return
	}

	cfg := getConfig()
	if err := printText(cfg, req.Text, req.Align, req.Bold, boolValue(req.Cut, true), boolValue(req.TrimTrailingBlank, cfg.TrimTrailingBlank)); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handlePrintRaw(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var err error

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			writeError(w, http.StatusBadRequest, "upload invalido")
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "campo file obrigatorio")
			return
		}
		defer file.Close()
		data, err = readAllLimited(file, maxUploadBytes)
	} else {
		var req printRawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "json invalido")
			return
		}
		data, err = decodeBase64Field(req.DataBase64)
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg := getConfig()
	if err := printRaw(cfg, data); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handlePrintPDF(w http.ResponseWriter, r *http.Request) {
	cutEnd := true
	cutBetweenPages := false
	trimTrailingBlank := false
	var pdfData []byte
	var err error

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			writeError(w, http.StatusBadRequest, "upload invalido")
			return
		}

		cfg := getConfig()
		if value := r.FormValue("cut"); value != "" {
			cutEnd = value == "true" || value == "1"
		}
		if value := r.FormValue("cut_between_pages"); value != "" {
			cutBetweenPages = value == "true" || value == "1"
		}
		trimTrailingBlank = trimTrailingBlankValue(r.FormValue("trim_trailing_blank"), nil, cfg.TrimTrailingBlank)

		file, _, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "campo file obrigatorio")
			return
		}
		defer file.Close()
		pdfData, err = readAllLimited(file, maxUploadBytes)
	} else {
		var req printPDFRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "json invalido")
			return
		}
		cfg := getConfig()
		cutEnd = boolValue(req.Cut, true)
		cutBetweenPages = boolValue(req.CutBetweenPages, false)
		trimTrailingBlank = boolValue(req.TrimTrailingBlank, cfg.TrimTrailingBlank)
		pdfData, err = decodeBase64Field(req.PDFBase64)
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg := getConfig()
	if err := printPDFBytes(cfg, pdfData, cutEnd, cutBetweenPages, trimTrailingBlank); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handlePrintImage(w http.ResponseWriter, r *http.Request) {
	cut := true
	trimTrailingBlank := false
	var imageData []byte
	var err error

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			writeError(w, http.StatusBadRequest, "upload invalido")
			return
		}

		cfg := getConfig()
		if value := r.FormValue("cut"); value != "" {
			cut = value == "true" || value == "1"
		}
		trimTrailingBlank = trimTrailingBlankValue(r.FormValue("trim_trailing_blank"), nil, cfg.TrimTrailingBlank)

		file, _, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "campo file obrigatorio")
			return
		}
		defer file.Close()
		imageData, err = readAllLimited(file, maxUploadBytes)
	} else {
		var req printImageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "json invalido")
			return
		}
		cfg := getConfig()
		cut = boolValue(req.Cut, true)
		trimTrailingBlank = boolValue(req.TrimTrailingBlank, cfg.TrimTrailingBlank)
		imageData, err = decodeBase64Field(req.ImageBase64)
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg := getConfig()
	if err := printImageBytes(cfg, imageData, cut, trimTrailingBlank); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type discoverApplyRequest struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func handleDiscover(w http.ResponseWriter, r *http.Request) {
	timeout := 400 * time.Millisecond
	if raw := r.URL.Query().Get("timeout_ms"); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms >= 100 && ms <= 3000 {
			timeout = time.Duration(ms) * time.Millisecond
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := discoverPrinters(ctx, timeout)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if r.URL.Query().Get("auto") == "true" && result.Count == 1 {
		printer := result.Printers[0]
		updated, err := updateConfig(Config{
			PrinterHost: printer.Host,
			PrinterPort: printer.Port,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"subnets":  result.Subnets,
			"printers": result.Printers,
			"count":    result.Count,
			"duration": result.Duration,
			"applied":  updated,
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func handleDiscoverApply(w http.ResponseWriter, r *http.Request) {
	var req discoverApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}

	if strings.TrimSpace(req.Host) == "" {
		writeError(w, http.StatusBadRequest, "campo host obrigatorio")
		return
	}
	if req.Port == 0 {
		req.Port = 9100
	}

	updated, err := updateConfig(Config{
		PrinterHost: req.Host,
		PrinterPort: req.Port,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"config": updated,
	})
}

func handlePrintTest(w http.ResponseWriter, r *http.Request) {
	cfg := getConfig()
	if err := printTestPage(cfg); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("pagina de teste enviada para %s", cfg.printerAddr()),
	})
}

func handlePrintBarcode(w http.ResponseWriter, r *http.Request) {
	var req printBarcodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if strings.TrimSpace(req.Data) == "" {
		writeError(w, http.StatusBadRequest, "campo data obrigatorio")
		return
	}

	cfg := getConfig()
	if err := printBarcode(cfg, req.Type, req.Data, req.Label, req.Align, boolValue(req.Cut, true)); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handlePrintQRCode(w http.ResponseWriter, r *http.Request) {
	var req printQRCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if strings.TrimSpace(req.Data) == "" {
		writeError(w, http.StatusBadRequest, "campo data obrigatorio")
		return
	}

	cfg := getConfig()
	if err := printQRCode(cfg, req.Data, req.Label, req.Align, boolValue(req.Cut, true)); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
