package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed web
var webFS embed.FS

func newRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", redirectToUI)
	mux.HandleFunc("GET /printit", handlePrintitIndex)
	mux.HandleFunc("GET /printit/health", handleHealth)
	mux.HandleFunc("GET /printit/status", handleStatus)
	mux.HandleFunc("PUT /printit/config", handlePutConfig)
	mux.HandleFunc("GET /printit/discover", handleDiscover)
	mux.HandleFunc("POST /printit/text", handlePrintText)
	mux.HandleFunc("POST /printit/raw", handlePrintRaw)
	mux.HandleFunc("POST /printit/pdf", handlePrintPDF)
	mux.HandleFunc("POST /printit/image", handlePrintImage)
	mux.HandleFunc("POST /printit/preview", handleFilePreview)
	mux.HandleFunc("POST /printit/test", handlePrintTest)
	mux.HandleFunc("POST /printit/reset", handlePrinterReset)
	mux.HandleFunc("POST /printit/barcode", handlePrintBarcode)
	mux.HandleFunc("POST /printit/qrcode", handlePrintQRCode)
	mux.HandleFunc("GET /printit/barcodes/preview", handleBarcodesPreview)

	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /printit/", http.StripPrefix("/printit/", webStaticHandler(webRoot)))

	return withCORS(mux)
}

func readWebAsset(relPath string) ([]byte, error) {
	if dir := webDevDir(); dir != "" {
		return os.ReadFile(filepath.Join(dir, relPath))
	}
	return fs.ReadFile(webFS, path.Join("web", relPath))
}

func webDevDir() string {
	if dir := os.Getenv("PRINT_IT_WEB_DIR"); dir != "" {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	if _, err := os.Stat("web/index.html"); err == nil {
		return "web"
	}

	return ""
}

func webStaticHandler(embedded fs.FS) http.Handler {
	if dir := webDevDir(); dir != "" {
		handler := http.FileServer(http.Dir(dir))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			handler.ServeHTTP(w, r)
		})
	}

	return http.FileServer(http.FS(embedded))
}

func redirectToUI(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/printit/", http.StatusTemporaryRedirect)
}

func handlePrintitIndex(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		http.Redirect(w, r, "/printit/", http.StatusTemporaryRedirect)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "/printit/status",
		"health": "/printit/health",
		"admin":  "/printit/",
	})
}
