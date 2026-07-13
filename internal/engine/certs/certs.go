package certs

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	sshlib "main/internal/ssh"
)

type Engine struct {
	client *sshlib.Client
}

func NewEngine(c *sshlib.Client) *Engine {
	return &Engine{client: c}
}

type CertInfo struct {
	Name       string
	Domains    []string
	ExpiryDate time.Time
	DaysLeft   int
	Issuer     string
	Valid      bool
}

func (e *Engine) ListCertificates() ([]CertInfo, error) {
	// First check if certbot is installed
	out, err := e.client.Run("certbot certificates")
	if err != nil {
		if strings.Contains(err.Error(), "command not found") || strings.Contains(err.Error(), "127") {
			return nil, fmt.Errorf("certbot is not installed on this server")
		}
		// Try with sudo
		out, err = e.client.Run("sudo certbot certificates")
		if err != nil {
			return nil, fmt.Errorf("failed to run certbot: %v", err)
		}
	}

	var certs []CertInfo
	var currentCert *CertInfo

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "Certificate Name:") {
			if currentCert != nil {
				certs = append(certs, *currentCert)
			}
			currentCert = &CertInfo{}
			currentCert.Name = strings.TrimSpace(strings.TrimPrefix(line, "Certificate Name:"))
		} else if currentCert != nil {
			if strings.HasPrefix(line, "Domains:") {
				domainsStr := strings.TrimSpace(strings.TrimPrefix(line, "Domains:"))
				currentCert.Domains = strings.Split(domainsStr, " ")
			} else if strings.HasPrefix(line, "Expiry Date:") {
				// Example: Expiry Date: 2024-05-15 12:00:00+00:00 (VALID: 89 days)
				expiryStr := strings.TrimSpace(strings.TrimPrefix(line, "Expiry Date:"))
				
				// Extract VALID/INVALID
				if strings.Contains(expiryStr, "VALID") {
					currentCert.Valid = true
				}
				
				// Extract days left
				re := regexp.MustCompile(`(?:VALID|INVALID):\s*(\d+)\s*days`)
				matches := re.FindStringSubmatch(expiryStr)
				if len(matches) > 1 {
					days, _ := strconv.Atoi(matches[1])
					currentCert.DaysLeft = days
				}
				
				// Try to parse the date up to the parenthesis
				idx := strings.Index(expiryStr, "(")
				if idx > 0 {
					dateStr := strings.TrimSpace(expiryStr[:idx])
					// 2024-05-15 12:00:00+00:00
					t, err := time.Parse("2006-01-02 15:04:05-07:00", dateStr)
					if err == nil {
						currentCert.ExpiryDate = t
					}
				}
			}
		}
	}
	if currentCert != nil {
		certs = append(certs, *currentCert)
	}

	// Sort by expiry days ascending
	sort.Slice(certs, func(i, j int) bool {
		return certs[i].DaysLeft < certs[j].DaysLeft
	})

	return certs, nil
}

func (e *Engine) RenewCertificate(name string) (string, error) {
	// Attempt renewal
	out, err := e.client.Run(fmt.Sprintf("sudo certbot renew --cert-name %s", name))
	if err != nil {
		// fallback to non-sudo
		out, err = e.client.Run(fmt.Sprintf("certbot renew --cert-name %s", name))
	}
	return out, err
}
