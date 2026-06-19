package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

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

func escposImageConfig(cfg Config, cut bool) *escposimg.Config {
	paperWidth := cfg.printableWidthMM()

	return &escposimg.Config{
		PaperWidthMM:  paperWidth,
		DPI:           escposimg.DPI203,
		DitheringAlgo: escposimg.DitheringFloydSteinberg,
		PrintMode:     escposimg.PrintModeRaster,
		CutPaper:      cut,
	}
}

func printText(cfg Config, text string, align string, bold bool, cut bool, trimTrailingBlank bool) error {
	if trimTrailingBlank {
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
		if err := printer.SetBold(true); err != nil {
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
		if err := printer.SetBold(false); err != nil {
			return err
		}
	}

	if cut {
		if err := printer.FeedLines(3); err != nil {
			return err
		}
		if err := printer.Cut(); err != nil {
			return err
		}
	}

	return nil
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

func printImageFile(cfg Config, imagePath string, cut bool) error {
	output, err := escposimg.NewNetworkOutput(cfg.printerAddr())
	if err != nil {
		return err
	}
	defer output.Close()

	return escposimg.ProcessImage(imagePath, escposImageConfig(cfg, cut), output)
}

func printImageBytes(cfg Config, imageData []byte, cut bool, trimTrailingBlank bool) error {
	tmpDir, err := os.MkdirTemp("", "printit-img-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	imagePath := filepath.Join(tmpDir, "input.png")
	if trimTrailingBlank {
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

func printPDFBytes(cfg Config, pdfData []byte, cutEnd bool, cutBetweenPages bool, trimTrailingBlank bool) error {
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

	imgCfg := escposImageConfig(cfg, false)

	for page := 0; page < doc.NumPage(); page++ {
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
		if trimTrailingBlank {
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

		pageCut := false
		if page < doc.NumPage()-1 {
			pageCut = cutBetweenPages
		} else {
			pageCut = cutEnd
		}
		pageCfg := *imgCfg
		pageCfg.CutPaper = pageCut

		output, err := escposimg.NewNetworkOutput(cfg.printerAddr())
		if err != nil {
			return fmt.Errorf("pagina %d: %w", page+1, err)
		}

		if err := escposimg.ProcessImage(pagePath, &pageCfg, output); err != nil {
			return fmt.Errorf("pagina %d: %w", page+1, err)
		}
	}

	return nil
}

func printTestPage(cfg Config) error {
	text := strings.Join([]string{
		"print.it - teste",
		"------------------------------",
		"Impressora: " + cfg.printerAddr(),
		"Papel: " + fmt.Sprintf("%dmm", cfg.PaperWidthMM),
		"",
		"Se voce leu isto,",
		"a conexao esta OK!",
		"",
	}, "\n")

	return printText(cfg, text, "center", false, true, false)
}

func printBarcode(cfg Config, barcodeType string, data string, label string, align string, cut bool) error {
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

	if cut {
		if err := printer.FeedLines(3); err != nil {
			return err
		}
		if err := printer.Cut(); err != nil {
			return err
		}
	}

	return nil
}

func printQRCode(cfg Config, data string, label string, align string, cut bool) error {
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

	if cut {
		if err := printer.FeedLines(3); err != nil {
			return err
		}
		if err := printer.Cut(); err != nil {
			return err
		}
	}

	return nil
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
