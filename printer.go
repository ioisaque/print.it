package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/72nd/escposimg"
	"github.com/gen2brain/go-fitz"
	"github.com/joeyak/go-escpos"
)

func dialPrinter(cfg Config) (net.Conn, error) {
	conn, err := net.Dial("tcp", cfg.printerAddr())
	if err != nil {
		return nil, fmt.Errorf("nao foi possivel conectar em %s: %w", cfg.printerAddr(), err)
	}
	return conn, nil
}

func probePrinterReachable(cfg Config) bool {
	if strings.TrimSpace(cfg.PrinterHost) == "" {
		return false
	}
	conn, err := net.DialTimeout("tcp", cfg.printerAddr(), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func resetPrinter(cfg Config) error {
	conn, err := dialPrinter(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.Write([]byte{0x18, 0x1B, '@'}); err != nil {
		return err
	}
	return escpos.NewPrinter(conn).FeedLines(2)
}

type CutMode string

const (
	CutNone    CutMode = "none"
	CutFull    CutMode = "full"
	CutPartial CutMode = "partial"
)

type TrimMode string

const (
	TrimNever    TrimMode = "never"
	TrimPage     TrimMode = "page"
	TrimDocument TrimMode = "document"
)

func parseCutMode(value string, fallback CutMode) CutMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "full", "total":
		return CutFull
	case "partial", "parcial":
		return CutPartial
	case "none", "nenhum":
		return CutNone
	default:
		return fallback
	}
}

func parseTrimMode(value string, fallback TrimMode) TrimMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "page", "pagina":
		return TrimPage
	case "document", "documento":
		return TrimDocument
	case "never", "nunca":
		return TrimNever
	default:
		return fallback
	}
}

func shouldTrimPage(mode TrimMode, pageIndex, totalPages int) bool {
	switch mode {
	case TrimPage:
		return true
	case TrimDocument:
		return pageIndex == totalPages-1
	default:
		return false
	}
}

func applyCut(printer escpos.Printer, mode CutMode) error {
	if mode == CutNone {
		return nil
	}
	if err := printer.FeedLines(3); err != nil {
		return err
	}
	if mode == CutFull {
		return printer.Cut()
	}
	_, err := printer.Write([]byte{0x1D, 'V', 1})
	return err
}

func sendCutCommand(cfg Config, mode CutMode) error {
	if mode == CutNone {
		return nil
	}
	conn, err := dialPrinter(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	return applyCut(escpos.NewPrinter(conn), mode)
}

func escposImageConfig(cfg Config) *escposimg.Config {
	paperWidth := cfg.printableWidthMM()

	return &escposimg.Config{
		PaperWidthMM:  paperWidth,
		DPI:           escposimg.DPI203,
		DitheringAlgo: escposimg.DitheringFloydSteinberg,
		PrintMode:     escposimg.PrintModeRaster,
		CutPaper:      false,
	}
}

func imageToGrayscalePNG(img image.Image, contrast int) ([]byte, error) {
	if contrast > 0 && contrast != 100 {
		img = adjustImageContrast(img, contrast)
	}
	gray := image.NewGray(img.Bounds())
	draw.Draw(gray, img.Bounds(), img, img.Bounds().Min, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, gray); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func adjustImageContrast(img image.Image, contrast int) image.Image {
	if contrast <= 0 || contrast == 100 {
		return img
	}

	factor := float64(contrast) / 100
	bounds := img.Bounds()
	out := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			g := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			v := (float64(g.Y)-128)*factor + 128
			if v < 0 {
				v = 0
			} else if v > 255 {
				v = 255
			}
			out.SetGray(x, y, color.Gray{Y: uint8(v + 0.5)})
		}
	}
	return out
}

func contrastProcessedImagePath(imagePath string, contrast int) (string, error) {
	if contrast <= 0 || contrast == 100 {
		return imagePath, nil
	}

	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("imagem invalida: %w", err)
	}

	tmp, err := os.CreateTemp("", "printit-contrast-*.png")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	if err := png.Encode(tmp, adjustImageContrast(img, contrast)); err != nil {
		tmp.Close()
		os.Remove(path)
		return "", err
	}
	tmp.Close()
	return path, nil
}

func filePreviewPNG(cfg Config, data []byte, filename string) ([]byte, error) {
	contrast := cfg.PrintContrast
	name := strings.ToLower(filename)
	if strings.HasSuffix(name, ".pdf") || (len(data) >= 4 && string(data[:4]) == "%PDF") {
		doc, err := fitz.NewFromMemory(data)
		if err != nil {
			return nil, fmt.Errorf("pdf invalido: %w", err)
		}
		defer doc.Close()

		if doc.NumPage() == 0 {
			return nil, fmt.Errorf("pdf sem paginas")
		}

		rgba, err := doc.ImageDPI(0, 120)
		if err != nil {
			return nil, err
		}

		return imageToGrayscalePNG(rgba, contrast)
	}

	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("imagem invalida: %w", err)
	}

	return imageToGrayscalePNG(decoded, contrast)
}

func printText(cfg Config, text string, align string, bold bool, cut CutMode, trim TrimMode) error {
	if trim != TrimNever {
		lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
		end := len(lines)
		for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
			end--
		}
		text = strings.Join(lines[:end], "\n")
	}

	conn, err := dialPrinter(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	printer := escpos.NewPrinter(conn)
	if err := printer.Initialize(); err != nil {
		return err
	}

	switch strings.ToLower(align) {
	case "center":
		if err := printer.Justify(escpos.CenterJustify); err != nil {
			return err
		}
	case "right":
		if err := printer.Justify(escpos.RightJustify); err != nil {
			return err
		}
	default:
		if err := printer.Justify(escpos.LeftJustify); err != nil {
			return err
		}
	}

	if bold {
		if err := printer.SelectPrintMode(escpos.Bold); err != nil {
			return err
		}
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for _, line := range lines {
		if err := printer.Println(line); err != nil {
			return err
		}
	}

	if bold {
		if err := printer.SelectPrintMode(); err != nil {
			return err
		}
	}

	return applyCut(printer, cut)
}

func printRaw(cfg Config, data []byte) error {
	conn, err := dialPrinter(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(data)
	return err
}

func printImageFile(cfg Config, imagePath string, cut CutMode) error {
	processedPath, err := contrastProcessedImagePath(imagePath, cfg.PrintContrast)
	if err != nil {
		return err
	}
	if processedPath != imagePath {
		defer os.Remove(processedPath)
	}

	output, err := escposimg.NewNetworkOutput(cfg.printerAddr())
	if err != nil {
		return err
	}

	if err := escposimg.ProcessImage(processedPath, escposImageConfig(cfg), output); err != nil {
		return err
	}
	return sendCutCommand(cfg, cut)
}

func printImageBytes(cfg Config, imageData []byte, cut CutMode, trim TrimMode) error {
	tmpDir, err := os.MkdirTemp("", "printit-img-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	imagePath := filepath.Join(tmpDir, "input.png")
	if trim != TrimNever {
		decoded, _, err := image.Decode(bytes.NewReader(imageData))
		if err != nil {
			return fmt.Errorf("imagem invalida: %w", err)
		}
		decoded = trimImageTrailingBlank(decoded)
		if decoded.Bounds().Dy() == 0 {
			return nil
		}
		file, err := os.Create(imagePath)
		if err != nil {
			return err
		}
		if err := png.Encode(file, decoded); err != nil {
			file.Close()
			return err
		}
		file.Close()
	} else if err := os.WriteFile(imagePath, imageData, 0o600); err != nil {
		return err
	}

	return printImageFile(cfg, imagePath, cut)
}

func printPDFBytes(cfg Config, pdfData []byte, cutAfterPage, cutAfterDoc CutMode, trim TrimMode) error {
	doc, err := fitz.NewFromMemory(pdfData)
	if err != nil {
		return fmt.Errorf("pdf invalido: %w", err)
	}
	defer doc.Close()

	if doc.NumPage() == 0 {
		return fmt.Errorf("pdf sem paginas")
	}

	tmpDir, err := os.MkdirTemp("", "printit-pdf-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	imgCfg := escposImageConfig(cfg)
	totalPages := doc.NumPage()

	for page := 0; page < totalPages; page++ {
		bounds, err := doc.Bound(page)
		if err != nil {
			return fmt.Errorf("pagina %d: %w", page+1, err)
		}

		pageWidthPts := bounds.Dx()
		if pageWidthPts <= 0 {
			return fmt.Errorf("pagina %d: largura invalida", page+1)
		}

		targetWidth := cfg.printablePixelWidth()
		renderDPI := float64(targetWidth) * 72.0 / float64(pageWidthPts)

		rgba, err := doc.ImageDPI(page, renderDPI)
		if err != nil {
			return fmt.Errorf("pagina %d: %w", page+1, err)
		}

		pageImg := image.Image(rgba)
		if shouldTrimPage(trim, page, totalPages) {
			pageImg = trimImageTrailingBlank(pageImg)
			if pageImg.Bounds().Dy() == 0 {
				continue
			}
		}

		pagePath := filepath.Join(tmpDir, fmt.Sprintf("page-%03d.png", page))
		file, err := os.Create(pagePath)
		if err != nil {
			return err
		}

		if err := png.Encode(file, pageImg); err != nil {
			file.Close()
			return err
		}
		file.Close()

		pageCut := cutAfterDoc
		if page < totalPages-1 {
			pageCut = cutAfterPage
		}

		output, err := escposimg.NewNetworkOutput(cfg.printerAddr())
		if err != nil {
			return fmt.Errorf("pagina %d: %w", page+1, err)
		}

		if err := escposimg.ProcessImage(pagePath, imgCfg, output); err != nil {
			return fmt.Errorf("pagina %d: %w", page+1, err)
		}
		if err := sendCutCommand(cfg, pageCut); err != nil {
			return fmt.Errorf("pagina %d: %w", page+1, err)
		}
	}

	return nil
}

func printTestPage(cfg Config, printer DiscoveredPrinter) error {
	if printer.Label == "" {
		printer.Label = friendlyPrinterLabel(printer)
	}

	lines := []string{
		"print.it",
		"------------------------------",
	}

	appendIfUseful := func(label, value string) {
		if isUsefulDeviceString(value) && !isJunkValue(value) {
			lines = append(lines, label+": "+value)
		}
	}

	appendIfUseful("Endereco", cfg.printerAddr())
	appendIfUseful("Nome", printer.Label)
	if printer.Name != "" && printer.Name != printer.Label {
		appendIfUseful("Identificacao", printer.Name)
	}
	appendIfUseful("Marca", printer.Manufacturer)
	appendIfUseful("Modelo", printer.Model)
	appendIfUseful("Serie", printer.Serial)
	appendIfUseful("Host", printer.Hostname)
	appendIfUseful("MAC", printer.MAC)
	appendIfUseful("Descricao", printer.Description)
	lines = append(lines, fmt.Sprintf("Papel: %dmm", cfg.PaperWidthMM))
	lines = append(lines, fmt.Sprintf("Area imprimivel: %dmm", cfg.printableWidthMM()))
	lines = append(lines, "", "Se voce leu isto,", "a conexao esta OK!", "")

	return printText(cfg, strings.Join(lines, "\n"), "center", false, CutPartial, TrimNever)
}

func printBarcode(cfg Config, barcodeType string, data string, label string, align string, cut CutMode) error {
	conn, err := dialPrinter(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	printer := escpos.NewPrinter(conn)
	if err := printer.Initialize(); err != nil {
		return err
	}

	switch strings.ToLower(align) {
	case "center":
		if err := printer.Justify(escpos.CenterJustify); err != nil {
			return err
		}
	case "right":
		if err := printer.Justify(escpos.RightJustify); err != nil {
			return err
		}
	default:
		if err := printer.Justify(escpos.LeftJustify); err != nil {
			return err
		}
	}

	if label != "" {
		if err := printer.Println(label); err != nil {
			return err
		}
	}

	bcType := escpos.BcCODE123
	switch strings.ToUpper(strings.ReplaceAll(barcodeType, "-", "")) {
	case "CODE39":
		bcType = escpos.BcCODE39
	case "EAN13", "JAN13":
		bcType = escpos.BcJAN13
	case "EAN8", "JAN8":
		bcType = escpos.BcJAN8
	case "UPCA":
		bcType = escpos.BcUPCA
	case "ITF":
		bcType = escpos.BcITF
	case "CODE93":
		bcType = escpos.BcCODE93
	}

	if err := printer.SetHRIPosition(escpos.HRIBelow); err != nil {
		return err
	}
	if err := printer.SetBarCodeHeight(80); err != nil {
		return err
	}
	if err := printer.PrintBarCode(bcType, data); err != nil {
		return err
	}
	if err := printer.FeedLines(2); err != nil {
		return err
	}

	return applyCut(printer, cut)
}

func printQRCode(cfg Config, data string, label string, align string, cut CutMode) error {
	conn, err := dialPrinter(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	printer := escpos.NewPrinter(conn)
	if err := printer.Initialize(); err != nil {
		return err
	}

	switch strings.ToLower(align) {
	case "center":
		if err := printer.Justify(escpos.CenterJustify); err != nil {
			return err
		}
	case "right":
		if err := printer.Justify(escpos.RightJustify); err != nil {
			return err
		}
	default:
		if err := printer.Justify(escpos.LeftJustify); err != nil {
			return err
		}
	}

	if label != "" {
		if err := printer.Println(label); err != nil {
			return err
		}
	}

	qr := buildQRCodeCommands(data)
	if _, err := printer.Write(qr); err != nil {
		return err
	}
	if err := printer.FeedLines(2); err != nil {
		return err
	}

	return applyCut(printer, cut)
}

func buildQRCodeCommands(content string) []byte {
	data := []byte(content)
	buf := make([]byte, 0, len(data)+64)

	buf = append(buf, 0x1d, 0x28, 0x6b, 0x04, 0x00, 0x31, 0x41, 0x32, 0x00)
	buf = append(buf, 0x1d, 0x28, 0x6b, 0x03, 0x00, 0x31, 0x43, 0x06)
	buf = append(buf, 0x1d, 0x28, 0x6b, 0x03, 0x00, 0x31, 0x45, 0x31)

	storeLen := len(data) + 3
	buf = append(buf, 0x1d, 0x28, 0x6b, byte(storeLen%256), byte(storeLen/256), 0x31, 0x50, 0x30)
	buf = append(buf, data...)

	buf = append(buf, 0x1d, 0x28, 0x6b, 0x03, 0x00, 0x31, 0x51, 0x30)

	return buf
}

const (
	blankPixelThreshold = 0xF0
	trimRowMinDensity   = 0.15
	trimBottomPaddingPx = 12
)

func rowDarkRatio(img image.Image, y int) float64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	if width == 0 || y < bounds.Min.Y || y >= bounds.Max.Y {
		return 0
	}

	dark := 0
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		gray := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
		if gray.Y < blankPixelThreshold {
			dark++
		}
	}

	return float64(dark) / float64(width)
}

func trimImageTrailingBlank(img image.Image) image.Image {
	bounds := img.Bounds()
	lastContent := bounds.Min.Y - 1

	for y := bounds.Max.Y - 1; y >= bounds.Min.Y; y-- {
		if rowDarkRatio(img, y) >= trimRowMinDensity {
			lastContent = y
			break
		}
	}

	if lastContent < bounds.Min.Y {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}

	bottom := lastContent + 1 + trimBottomPaddingPx
	if bottom > bounds.Max.Y {
		bottom = bounds.Max.Y
	}

	crop := image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bottom)
	if sub, ok := img.(interface{ SubImage(image.Rectangle) image.Image }); ok {
		return sub.SubImage(crop)
	}

	return img
}

func decodeBase64Field(value string) ([]byte, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, fmt.Errorf("campo base64 vazio")
	}

	if idx := strings.Index(raw, ","); idx >= 0 {
		raw = raw[idx+1:]
	}

	return base64.StdEncoding.DecodeString(raw)
}

func readAllLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("arquivo maior que %d bytes", maxBytes)
	}
	return data, nil
}
