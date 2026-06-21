package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxUploadBytes = 10 << 20 // 10 MB

type printTextRequest struct {
	Text              string `json:"text"`
	Cut               *bool  `json:"cut"`
	CutAfterDocument  string `json:"cut_after_document"`
	Align             string `json:"align"`
	Bold              bool   `json:"bold"`
	TrimTrailingBlank *bool  `json:"trim_trailing_blank"`
	TrimBlank         string `json:"trim_blank"`
}

type printRawRequest struct {
	DataBase64 string `json:"data_base64"`
}

type printPDFRequest struct {
	PDFBase64         string `json:"pdf_base64"`
	Cut               *bool  `json:"cut"`
	CutAfterPage      string `json:"cut_after_page"`
	CutAfterDocument  string `json:"cut_after_document"`
	CutBetweenPages   *bool  `json:"cut_between_pages"`
	TrimTrailingBlank *bool  `json:"trim_trailing_blank"`
	TrimBlank         string `json:"trim_blank"`
}

type printImageRequest struct {
	ImageBase64       string `json:"image_base64"`
	Cut               *bool  `json:"cut"`
	CutAfterDocument  string `json:"cut_after_document"`
	TrimTrailingBlank *bool  `json:"trim_trailing_blank"`
	TrimBlank         string `json:"trim_blank"`
}

type printBarcodeRequest struct {
	Type             string `json:"type"`
	Data             string `json:"data"`
	Label            string `json:"label"`
	Align            string `json:"align"`
	Cut              *bool  `json:"cut"`
	CutAfterDocument string `json:"cut_after_document"`
}

type printQRCodeRequest struct {
	Data             string `json:"data"`
	Label            string `json:"label"`
	Align            string `json:"align"`
	Cut              *bool  `json:"cut"`
	CutAfterDocument string `json:"cut_after_document"`
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

type printOptions struct {
	CutAfterPage CutMode
	CutAfterDoc  CutMode
	Trim         TrimMode
}

func cutAfterDocFromLegacy(cut *bool, partialDefault bool) CutMode {
	if cut == nil {
		if partialDefault {
			return CutPartial
		}
		return CutFull
	}
	if !*cut {
		return CutNone
	}
	if partialDefault {
		return CutPartial
	}
	return CutFull
}

func parsePrintOptionsForm(r *http.Request, cfg Config, partialDocDefault bool) printOptions {
	opts := printOptions{
		CutAfterPage: CutNone,
		CutAfterDoc:  cutAfterDocFromLegacy(nil, partialDocDefault),
		Trim:         trimModeFromLegacy(cfg.TrimTrailingBlank),
	}

	if value := r.FormValue("cut_after_page"); value != "" {
		opts.CutAfterPage = parseCutMode(value, CutNone)
	} else if value := r.FormValue("cut_between_pages"); value != "" {
		if value == "true" || value == "1" {
			opts.CutAfterPage = CutPartial
		}
	}

	if value := r.FormValue("cut_after_document"); value != "" {
		opts.CutAfterDoc = parseCutMode(value, CutNone)
	} else if value := r.FormValue("cut"); value != "" {
		cut := value == "true" || value == "1"
		opts.CutAfterDoc = cutAfterDocFromLegacy(&cut, partialDocDefault)
	}

	if value := r.FormValue("trim_blank"); value != "" {
		opts.Trim = parseTrimMode(value, TrimNever)
	} else {
		opts.Trim = trimModeFromLegacy(trimTrailingBlankValue(r.FormValue("trim_trailing_blank"), nil, cfg.TrimTrailingBlank))
	}

	return opts
}

func trimModeFromLegacy(blank bool) TrimMode {
	if blank {
		return TrimDocument
	}
	return TrimNever
}

func parsePrintOptionsJSON(cutAfterPage, cutAfterDoc, trimBlank string, cut *bool, cutBetween *bool, trim *bool, cfg Config, partialDocDefault bool) printOptions {
	opts := printOptions{
		CutAfterPage: CutNone,
		CutAfterDoc:  cutAfterDocFromLegacy(cut, partialDocDefault),
		Trim:         trimModeFromLegacy(cfg.TrimTrailingBlank),
	}

	if cutAfterPage != "" {
		opts.CutAfterPage = parseCutMode(cutAfterPage, CutNone)
	} else if cutBetween != nil && *cutBetween {
		opts.CutAfterPage = CutPartial
	}

	if cutAfterDoc != "" {
		opts.CutAfterDoc = parseCutMode(cutAfterDoc, CutNone)
	}

	if trimBlank != "" {
		opts.Trim = parseTrimMode(trimBlank, TrimNever)
	} else if trim != nil {
		opts.Trim = trimModeFromLegacy(*trim)
	}

	return opts
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"version": version,
		"service": "print.it",
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	cfg := getConfig()
	cfg.BarcodesAPIKey = ""
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "print.it",
		"version": version,
		"printer": cfg.printerAddr(),
		"config":  cfg,
	})
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
	if patch.BarcodesAPIKey != "" {
		cfg.BarcodesAPIKey = patch.BarcodesAPIKey
	}
	if flags.TrimTrailingBlank != nil {
		cfg.TrimTrailingBlank = *flags.TrimTrailingBlank
	}
	if len(patch.CorsOrigins) > 0 {
		cfg.CorsOrigins = patch.CorsOrigins
	}

	updated, err := saveFullConfig(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	updated.BarcodesAPIKey = ""
	writeJSON(w, http.StatusOK, updated)
}

func handleBarcodesPreview(w http.ResponseWriter, r *http.Request) {
	txt := r.URL.Query().Get("txt")
	if strings.TrimSpace(txt) == "" {
		writeError(w, http.StatusBadRequest, "campo txt obrigatorio")
		return
	}

	cfg := getConfig()
	if cfg.BarcodesAPIKey == "" {
		writeError(w, http.StatusBadGateway, "barcodes_api_key nao configurada")
		return
	}

	params := r.URL.Query()
	params.Set("logo", "false")
	params.Set("key", cfg.BarcodesAPIKey)

	upstream, err := http.Get("https://api.isaque.it/barcodes?" + params.Encode())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer upstream.Body.Close()

	if upstream.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(upstream.Body, 4096))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = upstream.Status
		}
		writeError(w, upstream.StatusCode, message)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.Copy(w, upstream.Body)
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
	opts := parsePrintOptionsJSON("", req.CutAfterDocument, req.TrimBlank, req.Cut, nil, req.TrimTrailingBlank, cfg, false)
	if err := printText(cfg, req.Text, req.Align, req.Bold, opts.CutAfterDoc, opts.Trim); err != nil {
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
	var opts printOptions
	var pdfData []byte
	var err error

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			writeError(w, http.StatusBadRequest, "upload invalido")
			return
		}

		cfg := getConfig()
		opts = parsePrintOptionsForm(r, cfg, true)

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
		opts = parsePrintOptionsJSON(req.CutAfterPage, req.CutAfterDocument, req.TrimBlank, req.Cut, req.CutBetweenPages, req.TrimTrailingBlank, cfg, true)
		pdfData, err = decodeBase64Field(req.PDFBase64)
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg := getConfig()
	if err := printPDFBytes(cfg, pdfData, opts.CutAfterPage, opts.CutAfterDoc, opts.Trim); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handlePrintImage(w http.ResponseWriter, r *http.Request) {
	var opts printOptions
	var imageData []byte
	var err error

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			writeError(w, http.StatusBadRequest, "upload invalido")
			return
		}

		cfg := getConfig()
		opts = parsePrintOptionsForm(r, cfg, true)

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
		opts = parsePrintOptionsJSON("", req.CutAfterDocument, req.TrimBlank, req.Cut, nil, req.TrimTrailingBlank, cfg, true)
		imageData, err = decodeBase64Field(req.ImageBase64)
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg := getConfig()
	if err := printImageBytes(cfg, imageData, opts.CutAfterDoc, opts.Trim); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleFilePreview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "upload invalido")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "campo file obrigatorio")
		return
	}
	defer file.Close()

	data, err := readAllLimited(file, maxUploadBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	pngData, err := filePreviewPNG(data, header.Filename)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(pngData)
}

func handleDiscover(w http.ResponseWriter, r *http.Request) {
	timeout := 400 * time.Millisecond
	if raw := r.URL.Query().Get("timeout_ms"); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms >= 100 && ms <= 3000 {
			timeout = time.Duration(ms) * time.Millisecond
		}
	}

	deep := r.URL.Query().Get("deep") == "true"
	scanTimeout := 30 * time.Second
	if deep {
		scanTimeout = 60 * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), scanTimeout)
	defer cancel()

	result, err := discoverPrinters(ctx, timeout, deep)
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
	opts := parsePrintOptionsJSON("", req.CutAfterDocument, "", req.Cut, nil, nil, cfg, false)
	if err := printBarcode(cfg, req.Type, req.Data, req.Label, req.Align, opts.CutAfterDoc); err != nil {
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
	opts := parsePrintOptionsJSON("", req.CutAfterDocument, "", req.Cut, nil, nil, cfg, false)
	if err := printQRCode(cfg, req.Data, req.Label, req.Align, opts.CutAfterDoc); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
