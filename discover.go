package main

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

type DiscoveredPrinter struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Service    string `json:"service"`
	Configured bool   `json:"configured"`
}

type DiscoverResult struct {
	Subnets  []string            `json:"subnets"`
	Printers []DiscoveredPrinter `json:"printers"`
	Count    int                 `json:"count"`
	Duration string              `json:"duration"`
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
