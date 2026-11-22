package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type Config struct {
	Interval int       `json:"interval_seconds"`
	Timeout  int       `json:"timeout_seconds"`
	APIUrl   string    `json:"api_url"`
	Targets  []Address `json:"targets"`
}

type Address struct {
	Type    string
	Address string
}

type CheckResult struct {
	Target  string  `json:"target"`
	Type    string  `json:"type"`
	Status  string  `json:"status"`
	Latency float64 `json:"latency"`

	HTTP *HTTPResult `json:"http,omitempty"`
	DNS  *DNSResult  `json:"dns,omitempty"`
	TLS  *TLSResult  `json:"tls,omitempty"`
}

type HTTPResult struct {
	StatusCode int `json:"status_code"`
}

type DNSResult struct {
	Records []string `json:"records"`
}

type TLSResult struct {
	Issuer        string   `json:"issuer"`
	NotAfter      string   `json:"not_after"`
	DaysRemaining int      `json:"days_remaining"`
	DNSNames      []string `json:"dns_names"`
}

func checkTCP(address string, timeout time.Duration) (time.Duration, error) {
	start := time.Now()

	conn, err := net.DialTimeout("tcp", address, timeout)
	latency := time.Since(start)

	if err != nil {
		return latency, err
	}

	defer conn.Close()

	return latency, nil
}

func checkHTTP(client *http.Client, address string) (int, string, time.Duration, error) {
	start := time.Now()

	conn, err := client.Get(address)
	latency := time.Since(start)
	status := "UP"

	if err != nil {
		return 0, "DOWN", latency, fmt.Errorf("Network error: %w", err)
	}

	defer conn.Body.Close()

	if conn.StatusCode < 200 || conn.StatusCode >= 300 {
		status = "DOWN"
	}

	return conn.StatusCode, status, latency, nil
}

func checkDNS(address string, timeout time.Duration) ([]string, time.Duration, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		timeout,
	)
	defer cancel()

	start := time.Now()

	records := []string{}
	resolve := net.DefaultResolver

	dns, err := resolve.LookupIPAddr(ctx, address)

	latency := time.Since(start)

	if err != nil {
		return records, latency, fmt.Errorf("DNS lookup error: %w", err)
	}

	for _, ip := range dns {
		records = append(records, ip.String())
	}

	return records, latency, nil
}

func checkTLS(address string, timeout time.Duration) (string, string, int, []string, time.Duration, error) {
	start := time.Now()
	dialer := &net.Dialer{
		Timeout: timeout,
	}
	config := &tls.Config{
		InsecureSkipVerify: false,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, config)
	latency := time.Since(start)

	if err != nil {
		return "", "", 0, nil, latency, fmt.Errorf("Failed to connect: %w", err)
	}

	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates

	if len(certs) == 0 {
		return "", "", 0, nil, latency, fmt.Errorf("No certificates provided")
	}

	leaf := certs[0]

	remainingDays := int(time.Until(leaf.NotAfter).Hours() / 24)

	return leaf.Issuer.CommonName, leaf.NotAfter.Format(time.RFC3339), remainingDays, leaf.DNSNames, latency, nil
}

func runCheck(result CheckResult, client *http.Client, apiurl string) error {
	data, err := json.Marshal(result)

	if err != nil {
		return fmt.Errorf("JSON error: %w", err)
	}

	reader := bytes.NewReader(data)

	req, err := http.NewRequest("POST", apiurl, reader)

	if err != nil {
		return fmt.Errorf("Request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("Response error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned: %s", resp.Status)
	}

	return nil
}

func main() {
	var config Config
	data, err := os.ReadFile("../config.json")

	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	err = json.Unmarshal(data, &config)

	if err != nil {
		log.Fatalf("Failed to read json data: %v", err)
	}

	if config.Interval <= 0 || config.Timeout <= 0 {
		log.Fatal("Interval and timeout must be positive")
	}

	parsed, err := url.ParseRequestURI(config.APIUrl)

	if err != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" {
		log.Fatal("Invalid API URL")
	}

	if len(config.Targets) == 0 {
		log.Fatal("Wrong target list")
	}

	addresses := config.Targets

	timeout := time.Duration(config.Timeout) * time.Second

	ticker := time.NewTicker(time.Duration(config.Interval) * time.Second)
	defer ticker.Stop()

	client := &http.Client{
		Timeout: timeout,
	}

	for {
		var wg sync.WaitGroup
		results := make(chan CheckResult, len(addresses))

		for _, address := range addresses {
			wg.Add(1)

			go func(address Address) {
				defer wg.Done()

				result := CheckResult{
					Target: address.Address,
					Status: "UP",
				}

				var latency time.Duration
				var err error

				switch address.Type {
				case "http", "https":
					var statusCode int
					var status string

					statusCode, status, latency, err = checkHTTP(client, address.Address)
					result.Status = status
					result.HTTP = &HTTPResult{
						StatusCode: statusCode,
					}

				case "tcp":
					latency, err = checkTCP(address.Address, timeout)

				case "dns":
					var records []string

					records, latency, err = checkDNS(address.Address, timeout)
					result.DNS = &DNSResult{
						Records: records,
					}

				case "tls":
					var issuer string
					var notAfter string
					var days int
					var dnsNames []string

					issuer, notAfter, days, dnsNames, latency, err = checkTLS(address.Address, timeout)

					if err == nil {
						result.TLS = &TLSResult{
							Issuer:        issuer,
							NotAfter:      notAfter,
							DaysRemaining: days,
							DNSNames:      dnsNames,
						}
						if days < 14 {
							log.Printf("WARNING: TLS certificate expires soon: %s", address.Address)
						}
					}

				default:
					log.Printf("WARNING: Unknown protocol type: %s", address.Type)
					return
				}

				result.Type = address.Type
				result.Latency = latency.Seconds() * 1000

				if err != nil {
					log.Printf("Check failed: %v", err)
					result.Status = "DOWN"
				}

				log.Printf("%s %s %s %.2fms", result.Type, result.Target, result.Status, result.Latency)

				results <- result
			}(address)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		for result := range results {
			err := runCheck(result, client, config.APIUrl)

			if err != nil {
				log.Printf("Failed to run check: %v", err)
			}
		}

		<-ticker.C
	}
}
