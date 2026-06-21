package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	PrinterHost  string   `json:"printer_host"`
	PrinterPort  int      `json:"printer_port"`
	PrinterMAC   string   `json:"printer_mac,omitempty"`
	ListenHost   string   `json:"listen_host"`
	ListenPort   int      `json:"listen_port"`
	PaperWidthMM       int      `json:"paper_width_mm"`
	PrintableWidthMM   int      `json:"printable_width_mm"`
	PrintContrast      int      `json:"print_contrast"`
	TrimTrailingBlank  bool     `json:"trim_trailing_blank"`
	BarcodesAPIKey     string   `json:"barcodes_api_key"`
	CorsOrigins        []string `json:"cors_origins"`
}

var (
	configMu sync.RWMutex
	config   Config
)

func defaultConfig() Config {
	cfg := Config{
		PrinterHost:  "192.168.1.201",
		PrinterPort:  9100,
		ListenHost:   "127.0.0.1",
		ListenPort:   9280,
		PaperWidthMM: 80,
		PrintContrast: 100,
		CorsOrigins:  []string{"*"},
	}
	if buildBarcodesAPIKey != "" {
		cfg.BarcodesAPIKey = buildBarcodesAPIKey
	}
	return cfg
}

func loadConfig() error {
	configMu.Lock()
	defer configMu.Unlock()

	config = defaultConfig()

	data, err := os.ReadFile(configFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return saveConfigLocked()
		}
		return err
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	normalizeConfigLocked()
	return saveConfigLocked()
}

func getConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return config
}

func updateConfig(patch Config) (Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	if patch.PrinterHost != "" {
		config.PrinterHost = patch.PrinterHost
	}
	if patch.PrinterPort > 0 {
		config.PrinterPort = patch.PrinterPort
	}
	if patch.PrinterMAC != "" {
		config.PrinterMAC = normalizeMAC(patch.PrinterMAC)
	}
	if patch.ListenHost != "" {
		config.ListenHost = patch.ListenHost
	}
	if patch.ListenPort > 0 {
		config.ListenPort = patch.ListenPort
	}
	if patch.PaperWidthMM > 0 {
		config.PaperWidthMM = patch.PaperWidthMM
	}
	if patch.PrintableWidthMM > 0 {
		config.PrintableWidthMM = patch.PrintableWidthMM
	}
	if patch.PrintContrast > 0 {
		config.PrintContrast = patch.PrintContrast
	}
	if patch.BarcodesAPIKey != "" {
		config.BarcodesAPIKey = patch.BarcodesAPIKey
	}
	if len(patch.CorsOrigins) > 0 {
		config.CorsOrigins = patch.CorsOrigins
	}

	normalizeConfigLocked()
	if err := saveConfigLocked(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func saveFullConfig(cfg Config) (Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	config = cfg
	normalizeConfigLocked()
	if err := saveConfigLocked(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func normalizeConfigLocked() {
	if config.PrinterHost == "" {
		config.PrinterHost = "192.168.1.201"
	}
	if config.PrinterPort == 0 {
		config.PrinterPort = 9100
	}
	if config.ListenHost == "" {
		config.ListenHost = "127.0.0.1"
	}
	if config.ListenPort == 0 {
		config.ListenPort = 9280
	}
	if config.PaperWidthMM == 0 {
		config.PaperWidthMM = 80
	}
	if config.PrintContrast <= 0 {
		config.PrintContrast = 100
	}
	if len(config.CorsOrigins) == 0 {
		config.CorsOrigins = []string{"*"}
	}
}

func saveConfigLocked() error {
	path := configFilePath()
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (c Config) printerAddr() string {
	return fmt.Sprintf("%s:%d", c.PrinterHost, c.PrinterPort)
}

func (c Config) listenAddr() string {
	return fmt.Sprintf("%s:%d", c.ListenHost, c.ListenPort)
}

func (c Config) printableWidthMM() int {
	if c.PrintableWidthMM > 0 {
		return c.PrintableWidthMM
	}
	if c.PaperWidthMM >= 80 {
		return 72
	}
	if c.PaperWidthMM > 0 && c.PaperWidthMM <= 58 {
		return 48
	}
	if c.PaperWidthMM > 0 {
		return c.PaperWidthMM
	}
	return 72
}

func (c Config) printablePixelWidth() int {
	dpi := 203
	inches := float64(c.printableWidthMM()) / 25.4
	return int(inches * float64(dpi))
}
