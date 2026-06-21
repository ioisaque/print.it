package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed web
var webFS embed.FS

func newRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", redirectToUI)
	mux.HandleFunc("GET /printit", handlePrintitIndex)
	mux.HandleFunc("GET /printit/status", handleStatus)
	mux.HandleFunc("PUT /printit/config", handlePutConfig)
	mux.HandleFunc("GET /printit/discover", handleDiscover)
	mux.HandleFunc("POST /printit/text", handlePrintText)
	mux.HandleFunc("POST /printit/raw", handlePrintRaw)
	mux.HandleFunc("POST /printit/pdf", handlePrintPDF)
	mux.HandleFunc("POST /printit/image", handlePrintImage)
	mux.HandleFunc("POST /printit/preview", handleFilePreview)
	mux.HandleFunc("POST /printit/test", handlePrintTest)
	mux.HandleFunc("POST /printit/barcode", handlePrintBarcode)
	mux.HandleFunc("POST /printit/qrcode", handlePrintQRCode)
	mux.HandleFunc("GET /printit/barcodes/preview", handleBarcodesPreview)

	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	static := http.FileServer(http.FS(webRoot))
	mux.Handle("GET /printit/", http.StripPrefix("/printit/", static))

	return withCORS(mux)
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
		"admin":  "/printit/",
	})
}
