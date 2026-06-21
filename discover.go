package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/gosnmp/gosnmp"
)

type DiscoveredPrinter struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Service      string `json:"service"`
	Label        string `json:"label"`
	DeviceType   string `json:"device_type"`
	Name         string `json:"name,omitempty"`
	Hostname     string `json:"hostname,omitempty"`
	MAC          string `json:"mac,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
	MacVendor    string `json:"mac_vendor,omitempty"`
	Model        string `json:"model,omitempty"`
	Serial       string `json:"serial,omitempty"`
	Description  string `json:"description,omitempty"`
	Configured   bool   `json:"configured"`
}

type DiscoverResult struct {
	Subnets  []string            `json:"subnets"`
	Printers []DiscoveredPrinter `json:"printers"`
	Count    int                 `json:"count"`
	Mode     string              `json:"mode"`
	Duration string              `json:"duration"`
}

type enrichOptions struct {
	snmp   bool
	escpos bool
	http   bool
	smb    bool
	dns    bool
	mac    bool
}

var quickEnrichOptions = enrichOptions{
	http:   true,
	smb:    true,
	mac:    true,
	escpos: true,
}

var deepEnrichOptions = enrichOptions{
	snmp:   true,
	escpos: true,
	http:   true,
	smb:    true,
	dns:    true,
	mac:    true,
}

var ouiVendors = map[string]string{
	"00:07:25": "Bematech",
	"00:11:62": "Epson",
	"00:17:C8": "Kyocera",
	"00:1B:82": "Star Micronics",
	"00:1E:8C": "Zebra",
	"00:23:7D": "Bixolon",
	"00:24:9B": "Citizen",
	"00:50:43": "Bematech",
	"00:50:C2": "Elgin",
	"00:80:77": "Brother",
	"00:90:A9": "Epson",
	"08:00:37": "Epson",
	"18:0C:AC": "HP",
	"3C:52:82": "HP",
	"58:38:79": "Zjiang",
	"64:EB:8C": "HP",
	"9C:93:4E": "Xerox",
	"B4:B6:76": "HP",
	"E0:BB:9E": "Gprinter",
	"E8:48:B8": "Canon",
	"F4:30:B9": "HP",
}

var bonjourServiceTypes = []string{
	"_ipp._tcp",
	"_ipps._tcp",
	"_airprint._tcp",
	"_printer._tcp",
	"_pdl-datastream._tcp",
}

type bonjourPrinterInfo struct {
	Name    string
	Service string
}

func discoverBonjourPrinters(ctx context.Context, timeout time.Duration) map[string]bonjourPrinterInfo {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := make(map[string]bonjourPrinterInfo)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, serviceType := range bonjourServiceTypes {
		wg.Add(1)
		go func(serviceType string) {
			defer wg.Done()

			resolver, err := zeroconf.NewResolver(nil)
			if err != nil {
				return
			}

			entries := make(chan *zeroconf.ServiceEntry)
			go func() {
				_ = resolver.Browse(ctx, serviceType, "local.", entries)
			}()

			for entry := range entries {
				if entry == nil {
					continue
				}

				name := sanitizeDeviceString(unescapeDNSInstance(strings.TrimSpace(entry.Instance)))
				if !isUsefulDeviceString(name) || isJunkValue(name) {
					continue
				}

				for _, ip := range entry.AddrIPv4 {
					ipStr := ip.String()
					mu.Lock()
					existing, ok := result[ipStr]
					if !ok || bonjourServicePriority(serviceType) > bonjourServicePriority(existing.Service) {
						result[ipStr] = bonjourPrinterInfo{Name: name, Service: serviceType}
					}
					mu.Unlock()
				}
			}
		}(serviceType)
	}

	wg.Wait()
	return result
}

func bonjourServicePriority(serviceType string) int {
	switch serviceType {
	case "_ipp._tcp", "_ipps._tcp":
		return 4
	case "_airprint._tcp":
		return 3
	case "_printer._tcp":
		return 2
	case "_pdl-datastream._tcp":
		return 1
	default:
		return 0
	}
}

func applyBonjourInfo(printer *DiscoveredPrinter, info bonjourPrinterInfo) {
	setIfUseful(&printer.Name, info.Name)
	if printer.Manufacturer == "" {
		for _, brand := range []string{"Canon", "HP", "Epson", "Brother", "Kyocera", "Samsung", "Lexmark", "Ricoh", "Xerox"} {
			if strings.HasPrefix(strings.ToLower(info.Name), strings.ToLower(brand)) {
				printer.Manufacturer = brand
				rest := strings.TrimSpace(strings.TrimPrefix(info.Name, brand))
				setIfUseful(&printer.Model, rest)
				break
			}
		}
	}
	if info.Service != "" {
		printer.DeviceType = "Impressora (AirPrint/IPP)"
	}
}

func discoverPrinters(ctx context.Context, timeout time.Duration, deep bool) (DiscoverResult, error) {
	started := time.Now()
	mode := "quick"
	options := quickEnrichOptions
	enrichTimeout := timeout * 2

	if deep {
		mode = "deep"
		options = deepEnrichOptions
		enrichTimeout = timeout * 4
	}

	subnets, err := localSubnets24()
	if err != nil {
		return DiscoverResult{}, err
	}

	if len(subnets) == 0 {
		return DiscoverResult{}, fmt.Errorf("nenhuma interface de rede IPv4 ativa encontrada")
	}

	cfg := getConfig()
	targets := hostsForSubnets(subnets)
	found := make(map[string]DiscoveredPrinter)
	var mu sync.Mutex

	bonjourTimeout := timeout + 2*time.Second
	if bonjourTimeout > 4*time.Second {
		bonjourTimeout = 4 * time.Second
	}
	bonjourCh := make(chan map[string]bonjourPrinterInfo, 1)
	go func() {
		bonjourCh <- discoverBonjourPrinters(ctx, bonjourTimeout)
	}()

	sem := make(chan struct{}, 64)
	var wg sync.WaitGroup

	for _, host := range targets {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(host string) {
			defer wg.Done()
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			addr := net.JoinHostPort(host, "9100")
			dialer := net.Dialer{Timeout: timeout}
			conn, err := dialer.DialContext(ctx, "tcp", addr)
			if err != nil {
				return
			}
			conn.Close()

			mu.Lock()
			found[host] = DiscoveredPrinter{
				Host:       host,
				Port:       9100,
				Service:    "raw",
				Configured: false,
			}
			mu.Unlock()
		}(host)
	}

	wg.Wait()

	bonjourMap := <-bonjourCh
	for host, info := range bonjourMap {
		printer, ok := found[host]
		if !ok {
			continue
		}
		applyBonjourInfo(&printer, info)
		found[host] = printer
	}

	if len(found) > 0 {
		enrichSem := make(chan struct{}, 8)
		var enrichWg sync.WaitGroup

		for host, printer := range found {
			if ctx.Err() != nil {
				break
			}

			enrichWg.Add(1)
			enrichSem <- struct{}{}

			go func(host string, printer DiscoveredPrinter) {
				defer enrichWg.Done()
				defer func() { <-enrichSem }()

				enriched := enrichDiscoveredPrinter(ctx, host, printer, enrichTimeout, options)

				mu.Lock()
				found[host] = enriched
				mu.Unlock()
			}(host, printer)
		}

		enrichWg.Wait()
	}

	for host, printer := range found {
		if printer.DeviceType == "" {
			printer.DeviceType = "Impressora térmica"
		}
		if printer.Label == "" {
			printer.Label = friendlyPrinterLabel(printer)
		}
		printer.Configured = printerMatchesConfig(printer, cfg)
		found[host] = printer
	}

	printers := make([]DiscoveredPrinter, 0, len(found))
	for _, printer := range found {
		printers = append(printers, printer)
	}

	sort.Slice(printers, func(i, j int) bool {
		if printers[i].Configured != printers[j].Configured {
			return printers[i].Configured
		}
		return printers[i].Host < printers[j].Host
	})

	if !deep {
		printers = filterConfidentPrinters(printers)
	}

	subnetLabels := make([]string, 0, len(subnets))
	for _, subnet := range subnets {
		subnetLabels = append(subnetLabels, subnet.String())
	}

	return DiscoverResult{
		Subnets:  subnetLabels,
		Printers: printers,
		Count:    len(printers),
		Mode:     mode,
		Duration: time.Since(started).Round(time.Millisecond).String(),
	}, nil
}

func filterConfidentPrinters(printers []DiscoveredPrinter) []DiscoveredPrinter {
	filtered := make([]DiscoveredPrinter, 0, len(printers))
	for _, printer := range printers {
		if isConfidentPrinter(printer) {
			filtered = append(filtered, printer)
		}
	}
	return filtered
}

func isConfidentPrinter(printer DiscoveredPrinter) bool {
	if printer.Configured {
		return true
	}
	if printer.Host != "" && printer.Port == 9100 {
		return true
	}
	if isUsefulDeviceString(printer.Name) && !isJunkValue(printer.Name) {
		return true
	}
	if isUsefulDeviceString(printer.Manufacturer) && !isJunkValue(printer.Manufacturer) &&
		isUsefulDeviceString(printer.Model) && !isJunkValue(printer.Model) {
		return true
	}
	return false
}

func isJunkValue(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	switch lower {
	case "", "<nil>", "nil", "unknown", "n/a", "bsa/1.2":
		return true
	}
	if strings.HasPrefix(lower, "bsa/") {
		return true
	}
	if strings.HasPrefix(lower, "http/") {
		return true
	}
	if strings.Contains(lower, "404") && strings.Contains(lower, "not found") {
		return true
	}
	return false
}

func enrichDiscoveredPrinter(ctx context.Context, host string, printer DiscoveredPrinter, timeout time.Duration, options enrichOptions) DiscoveredPrinter {
	if ctx.Err() != nil {
		if printer.DeviceType == "" {
			printer.DeviceType = "Impressora térmica"
		}
		printer.Label = friendlyPrinterLabel(printer)
		return printer
	}

	if options.http {
		httpTitle, httpServer := probeWebInterface(host, timeout)
		setIfUseful(&printer.Name, httpTitle)
		if printer.Model == "" && isUsefulDeviceString(httpServer) && !isJunkValue(httpServer) {
			printer.Model = httpServer
		}
	}

	if options.smb {
		setIfUseful(&printer.Name, lookupSMBName(host))
	}

	if options.dns {
		printer.Hostname = lookupHostname(ctx, host)
	}

	if options.mac {
		ensureARP(host, timeout)
		printer.MAC = lookupMAC(host)
		if macVendor := vendorFromMAC(printer.MAC); macVendor != "" {
			printer.MacVendor = macVendor
		}
	}

	if options.snmp {
		snmpInfo := querySNMPPrinter(host, timeout)
		setIfUseful(&printer.Name, snmpInfo.Name)
		setIfUseful(&printer.Description, snmpInfo.Description)
		setIfUseful(&printer.Manufacturer, snmpInfo.Manufacturer)
		setIfUseful(&printer.Model, snmpInfo.Model)
		setIfUseful(&printer.Serial, snmpInfo.Serial)
	}

	if options.escpos {
		model, manufacturer, serial := queryPrinterIdentity(host, timeout)
		setIfUseful(&printer.Model, model)
		setIfUseful(&printer.Manufacturer, manufacturer)
		setIfUseful(&printer.Serial, serial)
	}

	if printer.Name == "" && isUsefulDeviceString(printer.Hostname) {
		printer.Name = printer.Hostname
	}

	if printer.DeviceType == "" {
		printer.DeviceType = "Impressora térmica"
	}
	printer.Label = friendlyPrinterLabel(printer)

	return printer
}

func ensureARP(host string, timeout time.Duration) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, "9100"), timeout)
	if err != nil {
		return
	}
	conn.Close()
}

func setIfUseful(dest *string, value string) {
	if *dest == "" && isUsefulDeviceString(value) && !isJunkValue(value) {
		*dest = value
	}
}

func friendlyPrinterLabel(printer DiscoveredPrinter) string {
	if isUsefulDeviceString(printer.Name) && !isJunkValue(printer.Name) {
		return printer.Name
	}

	if isUsefulDeviceString(printer.Manufacturer) && !isJunkValue(printer.Manufacturer) &&
		isUsefulDeviceString(printer.Model) && !isJunkValue(printer.Model) {
		return printer.Manufacturer + " " + printer.Model
	}

	if isUsefulDeviceString(printer.Manufacturer) && !isJunkValue(printer.Manufacturer) {
		return printer.Manufacturer
	}

	if isUsefulDeviceString(printer.Model) && !isJunkValue(printer.Model) {
		return printer.Model
	}

	if isUsefulDeviceString(printer.Hostname) && !isJunkValue(printer.Hostname) {
		return printer.Hostname
	}

	if isUsefulDeviceString(printer.MacVendor) && !isJunkValue(printer.MacVendor) && printer.Host != "" {
		return printer.MacVendor + " (" + printer.Host + ")"
	}

	if printer.Host != "" {
		return "Impressora " + printer.Host
	}

	return "Impressora não identificada"
}

func isUsefulDeviceString(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return false
	}

	if len(value) < 3 {
		for _, r := range value {
			if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
		}
		return true
	}

	alnum := 0
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == ' ', r == '-', r == '_', r == '.', r == '/':
			alnum++
		}
	}

	return alnum >= 2 && float64(alnum)/float64(len(value)) >= 0.5
}

func lookupSMBName(host string) string {
	if runtime.GOOS != "darwin" {
		return ""
	}

	out, err := exec.Command("smbutil", "lookup", "-w", host).Output()
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(string(out))
	if idx := strings.Index(line, "("); idx >= 0 {
		rest := line[idx+1:]
		if end := strings.Index(rest, ")"); end >= 0 {
			name := strings.TrimSpace(rest[:end])
			if isUsefulDeviceString(name) {
				return name
			}
		}
	}

	return ""
}

var htmlTitlePattern = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

func probeWebInterface(host string, timeout time.Duration) (title, server string) {
	client := &http.Client{Timeout: timeout}

	for _, path := range []string{"/", "/index.htm", "/index.html"} {
		response, err := client.Get("http://" + net.JoinHostPort(host, "80") + path)
		if err != nil {
			continue
		}

		body, _ := io.ReadAll(io.LimitReader(response.Body, 8192))
		response.Body.Close()

		server = strings.TrimSpace(response.Header.Get("Server"))
		if match := htmlTitlePattern.FindSubmatch(body); len(match) > 1 {
			title = sanitizeDeviceString(string(match[1]))
		}

		if isUsefulDeviceString(title) || isUsefulDeviceString(server) {
			return title, server
		}
	}

	return "", ""
}

type snmpPrinterInfo struct {
	Name         string
	Description  string
	Manufacturer string
	Model        string
	Serial       string
}

func querySNMPPrinter(host string, timeout time.Duration) snmpPrinterInfo {
	communities := []string{"public", "private"}
	oids := []string{
		"1.3.6.1.2.1.1.1.0",         // sysDescr
		"1.3.6.1.2.1.1.5.0",         // sysName
		"1.3.6.1.2.1.43.5.1.1.16.1", // prtGeneralPrinterName
		"1.3.6.1.2.1.43.5.1.1.17.1", // prtGeneralSerialNumber
	}

	for _, community := range communities {
		client := &gosnmp.GoSNMP{
			Target:    host,
			Port:      161,
			Community: community,
			Version:   gosnmp.Version2c,
			Timeout:   timeout,
			Retries:   1,
		}

		if err := client.Connect(); err != nil {
			continue
		}

		result, err := client.Get(oids)
		client.Conn.Close()
		if err != nil {
			continue
		}

		info := snmpPrinterInfo{}
		for _, variable := range result.Variables {
			value := snmpStringValue(variable)
			if value == "" {
				continue
			}

			switch variable.Name {
			case ".1.3.6.1.2.1.1.1.0":
				info.Description = value
				manufacturer, model := parseSysDescr(value)
				if info.Manufacturer == "" {
					info.Manufacturer = manufacturer
				}
				if info.Model == "" {
					info.Model = model
				}
			case ".1.3.6.1.2.1.1.5.0":
				if info.Name == "" {
					info.Name = value
				}
			case ".1.3.6.1.2.1.43.5.1.1.16.1":
				info.Name = value
			case ".1.3.6.1.2.1.43.5.1.1.17.1":
				info.Serial = value
			}
		}

		if info.Name != "" || info.Description != "" || info.Model != "" || info.Manufacturer != "" || info.Serial != "" {
			return info
		}
	}

	return snmpPrinterInfo{}
}

func snmpStringValue(variable gosnmp.SnmpPDU) string {
	switch variable.Type {
	case gosnmp.OctetString:
		if value, ok := variable.Value.([]byte); ok {
			return sanitizeDeviceString(string(value))
		}
	case gosnmp.ObjectIdentifier:
		if value, ok := variable.Value.(string); ok {
			return sanitizeDeviceString(value)
		}
	default:
		if variable.Value == nil {
			return ""
		}
		return sanitizeDeviceString(fmt.Sprint(variable.Value))
	}
	return ""
}

func parseSysDescr(descr string) (manufacturer, model string) {
	descr = strings.TrimSpace(descr)
	if descr == "" {
		return "", ""
	}

	parts := strings.Fields(descr)
	if len(parts) == 0 {
		return "", ""
	}

	manufacturer = parts[0]
	if len(parts) > 1 {
		model = strings.Join(parts[1:], " ")
	}

	return manufacturer, model
}

func lookupHostname(ctx context.Context, host string) string {
	resolver := net.Resolver{}
	names, err := resolver.LookupAddr(ctx, host)
	if err != nil || len(names) == 0 {
		return ""
	}

	hostname := strings.TrimSuffix(names[0], ".")
	if hostname == host {
		return ""
	}

	return hostname
}

func lookupMAC(host string) string {
	out, err := exec.Command("arp", "-n", host).Output()
	if err != nil {
		return ""
	}

	line := string(out)
	if idx := strings.Index(line, " at "); idx >= 0 {
		rest := line[idx+4:]
		if end := strings.Index(rest, " "); end >= 0 {
			mac := strings.ToLower(rest[:end])
			if mac != "(incomplete)" {
				return normalizeMAC(mac)
			}
		}
	}

	return ""
}

var arpLinePattern = regexp.MustCompile(`\(([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)\) at ([0-9a-f:]+)`)

func lookupIPByMAC(mac string) string {
	mac = normalizeMAC(mac)
	if mac == "" {
		return ""
	}

	out, err := exec.Command("arp", "-a").Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		match := arpLinePattern.FindStringSubmatch(line)
		if len(match) < 3 {
			continue
		}
		if normalizeMAC(match[2]) == mac {
			return match[1]
		}
	}

	return ""
}

func isHostPortOpen(host string, port int, timeout time.Duration) bool {
	if strings.TrimSpace(host) == "" {
		return false
	}
	if port == 0 {
		port = 9100
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func updatePrinterHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	cfg := getConfig()
	if cfg.PrinterHost == host {
		return false
	}
	cfg.PrinterHost = host
	if _, err := saveFullConfig(cfg); err != nil {
		return false
	}
	return true
}

func vendorFromMAC(mac string) string {
	mac = normalizeMAC(mac)
	prefix := strings.ToUpper(mac)
	if len(prefix) >= 8 {
		prefix = prefix[:8]
	}
	return ouiVendors[prefix]
}

func queryPrinterIdentity(host string, timeout time.Duration) (model, manufacturer, serial string) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, "9100"), timeout)
	if err != nil {
		return
	}
	defer conn.Close()

	deadline := time.Now().Add(timeout)
	_ = conn.SetDeadline(deadline)

	_, _ = conn.Write([]byte{0x1B, 0x40})

	queries := []struct {
		code byte
		dest *string
	}{
		{1, &model},
		{2, &manufacturer},
		{3, &serial},
	}

	for _, query := range queries {
		if time.Now().After(deadline) {
			break
		}

		_, err := conn.Write([]byte{0x1D, 0x49, query.code})
		if err != nil {
			continue
		}

		buf := make([]byte, 256)
		n, err := conn.Read(buf)
		if err != nil || n == 0 {
			continue
		}

		value := sanitizeDeviceString(string(buf[:n]))
		if isUsefulDeviceString(value) {
			*query.dest = value
		}
	}

	return
}

func sanitizeDeviceString(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= 32 && r < 127 {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func unescapeDNSInstance(value string) string {
	if !strings.Contains(value, `\`) {
		return value
	}
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			b.WriteByte(value[i+1])
			i++
			continue
		}
		b.WriteByte(value[i])
	}
	return b.String()
}

func localSubnets24() ([]*net.IPNet, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	subnets := make([]*net.IPNet, 0)

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}

			ip4 := ipNet.IP.To4()
			subnet := &net.IPNet{
				IP:   net.IPv4(ip4[0], ip4[1], ip4[2], 0),
				Mask: net.CIDRMask(24, 32),
			}

			key := subnet.String()
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			subnets = append(subnets, subnet)
		}
	}

	return subnets, nil
}

func hostsForSubnets(subnets []*net.IPNet) []string {
	seen := make(map[string]struct{})
	hosts := make([]string, 0)

	for _, subnet := range subnets {
		ip4 := subnet.IP.To4()
		if ip4 == nil {
			continue
		}

		for host := 1; host <= 254; host++ {
			ip := net.IPv4(ip4[0], ip4[1], ip4[2], byte(host)).String()
			if _, exists := seen[ip]; exists {
				continue
			}
			seen[ip] = struct{}{}
			hosts = append(hosts, ip)
		}
	}

	return hosts
}

func normalizeMAC(mac string) string {
	mac = strings.TrimSpace(strings.ToLower(mac))
	if mac == "" {
		return ""
	}
	mac = strings.ReplaceAll(mac, "-", ":")
	if !strings.Contains(mac, ":") && len(mac) == 12 {
		parts := make([]string, 0, 6)
		for i := 0; i < 12; i += 2 {
			parts = append(parts, mac[i:i+2])
		}
		mac = strings.Join(parts, ":")
	}

	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return mac
	}

	for i, part := range parts {
		if len(part) == 0 {
			parts[i] = "00"
			continue
		}
		if len(part) == 1 {
			part = "0" + part
		}
		if len(part) > 2 {
			part = part[len(part)-2:]
		}
		parts[i] = part
	}

	return strings.Join(parts, ":")
}

func printerMatchesConfig(printer DiscoveredPrinter, cfg Config) bool {
	port := cfg.PrinterPort
	if port == 0 {
		port = 9100
	}
	if printer.Port == 0 {
		printer.Port = 9100
	}

	if mac := normalizeMAC(cfg.PrinterMAC); mac != "" && normalizeMAC(printer.MAC) == mac {
		return printer.Port == port
	}

	return printer.Host == cfg.PrinterHost && printer.Port == port
}

func resolveConfiguredPrinter() bool {
	cfg := getConfig()
	mac := normalizeMAC(cfg.PrinterMAC)
	if mac == "" {
		return false
	}

	port := cfg.PrinterPort
	if port == 0 {
		port = 9100
	}

	if isHostPortOpen(cfg.PrinterHost, port, 800*time.Millisecond) {
		ensureARP(cfg.PrinterHost, 800*time.Millisecond)
		if normalizeMAC(lookupMAC(cfg.PrinterHost)) == mac {
			return false
		}
	}

	if ip := lookupIPByMAC(mac); ip != "" && ip != cfg.PrinterHost {
		if isHostPortOpen(ip, port, 800*time.Millisecond) {
			log.Printf("impressora encontrada na tabela ARP: %s (MAC %s)", ip, mac)
			return updatePrinterHost(ip)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	host, ok := findHostByMAC(ctx, mac, port, 400*time.Millisecond)
	if !ok || host == "" {
		return false
	}
	if host == cfg.PrinterHost {
		return false
	}

	log.Printf("impressora encontrada na rede: %s (MAC %s)", host, mac)
	return updatePrinterHost(host)
}

func findHostByMAC(ctx context.Context, mac string, port int, timeout time.Duration) (string, bool) {
	mac = normalizeMAC(mac)
	if mac == "" {
		return "", false
	}
	if port == 0 {
		port = 9100
	}

	subnets, err := localSubnets24()
	if err != nil {
		return "", false
	}

	targets := hostsForSubnets(subnets)
	var (
		mu         sync.Mutex
		foundHost  string
		sem        = make(chan struct{}, 64)
		wg         sync.WaitGroup
	)

	for _, host := range targets {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			mu.Lock()
			alreadyFound := foundHost != ""
			mu.Unlock()
			if alreadyFound {
				return
			}

			addr := net.JoinHostPort(host, strconv.Itoa(port))
			conn, err := net.DialTimeout("tcp", addr, timeout)
			if err != nil {
				return
			}
			conn.Close()

			ensureARP(host, timeout)
			if normalizeMAC(lookupMAC(host)) != mac {
				return
			}

			mu.Lock()
			if foundHost == "" {
				foundHost = host
			}
			mu.Unlock()
		}(host)
	}

	wg.Wait()
	if foundHost == "" {
		return "", false
	}
	return foundHost, true
}
