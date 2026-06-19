package main

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

type DiscoveredPrinter struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Service      string `json:"service"`
	Hostname     string `json:"hostname,omitempty"`
	MAC          string `json:"mac,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Model        string `json:"model,omitempty"`
	Serial       string `json:"serial,omitempty"`
	Configured   bool   `json:"configured"`
}

type DiscoverResult struct {
	Subnets  []string            `json:"subnets"`
	Printers []DiscoveredPrinter `json:"printers"`
	Count    int                 `json:"count"`
	Duration string              `json:"duration"`
}

var ouiVendors = map[string]string{
	"00:1B:82": "Star Micronics",
	"00:11:62": "Epson",
	"00:17:C8": "Kyocera",
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
	"64:EB:8C": "HP",
	"9C:93:4E": "Xerox",
	"B4:B6:76": "HP",
	"E8:48:B8": "Canon",
	"F4:30:B9": "HP",
}

func discoverPrinters(ctx context.Context, timeout time.Duration) (DiscoverResult, error) {
	started := time.Now()

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
				Configured: host == cfg.PrinterHost && cfg.PrinterPort == 9100,
			}
			mu.Unlock()
		}(host)
	}

	wg.Wait()

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

				enriched := enrichDiscoveredPrinter(ctx, host, printer, timeout*4)

				mu.Lock()
				found[host] = enriched
				mu.Unlock()
			}(host, printer)
		}

		enrichWg.Wait()
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

	subnetLabels := make([]string, 0, len(subnets))
	for _, subnet := range subnets {
		subnetLabels = append(subnetLabels, subnet.String())
	}

	return DiscoverResult{
		Subnets:  subnetLabels,
		Printers: printers,
		Count:    len(printers),
		Duration: time.Since(started).Round(time.Millisecond).String(),
	}, nil
}

func enrichDiscoveredPrinter(ctx context.Context, host string, printer DiscoveredPrinter, timeout time.Duration) DiscoveredPrinter {
	if ctx.Err() != nil {
		return printer
	}

	printer.Hostname = lookupHostname(ctx, host)
	printer.MAC = lookupMAC(host)

	if printer.MAC != "" && printer.Manufacturer == "" {
		printer.Manufacturer = vendorFromMAC(printer.MAC)
	}

	model, manufacturer, serial := queryPrinterIdentity(host, timeout)
	if model != "" {
		printer.Model = model
	}
	if manufacturer != "" {
		printer.Manufacturer = manufacturer
	}
	if serial != "" {
		printer.Serial = serial
	}

	return printer
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
				return mac
			}
		}
	}

	return ""
}

func vendorFromMAC(mac string) string {
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
		if value != "" {
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
