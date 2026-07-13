package proxy

import (
	"fmt"
	"strings"

	sshlib "main/internal/ssh"
)

type Engine struct {
	client *sshlib.Client
}

func NewEngine(c *sshlib.Client) *Engine {
	return &Engine{client: c}
}

type ProxyType string

const (
	ProxyNginx  ProxyType = "nginx"
	ProxyApache ProxyType = "apache"
	ProxyCaddy  ProxyType = "caddy"
	ProxyNone   ProxyType = "none"
)

type Site struct {
	Name      string
	Target    string
	SSLStatus string
	ConfPath  string
}

func (e *Engine) DetectProxy() ProxyType {
	if out, _ := e.client.Run("which nginx"); out != "" {
		return ProxyNginx
	}
	if out, _ := e.client.Run("which caddy"); out != "" {
		return ProxyCaddy
	}
	if out, _ := e.client.Run("which apache2"); out != "" {
		return ProxyApache
	}
	if out, _ := e.client.Run("which httpd"); out != "" {
		return ProxyApache
	}
	return ProxyNone
}

func (e *Engine) ListSites(ptype ProxyType) ([]Site, error) {
	var sites []Site
	if ptype == ProxyNginx {
		out, err := e.client.Run("ls -1 /etc/nginx/sites-enabled/")
		if err != nil || strings.Contains(out, "No such file") {
			out, _ = e.client.Run("ls -1 /etc/nginx/conf.d/")
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		for _, l := range lines {
			if l == "" || strings.Contains(l, "No such file") {
				continue
			}
			confPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s", l)
			// fallback check if using conf.d
			if strings.Contains(out, ".conf") {
				confPath = fmt.Sprintf("/etc/nginx/conf.d/%s", l)
			}
			content, _ := e.client.Run(fmt.Sprintf("cat %q", confPath))
			site := Site{Name: l, ConfPath: confPath, SSLStatus: "Inactive", Target: "Static"}
			if strings.Contains(content, "ssl_certificate") || strings.Contains(content, "listen 443 ssl") {
				site.SSLStatus = "Active"
			}
			if strings.Contains(content, "proxy_pass") {
				site.Target = "Proxy"
			}
			sites = append(sites, site)
		}
	} else if ptype == ProxyCaddy {
		content, err := e.client.Run("cat /etc/caddy/Caddyfile")
		if err == nil && !strings.Contains(content, "No such file") {
			site := Site{Name: "Caddyfile", ConfPath: "/etc/caddy/Caddyfile", SSLStatus: "Auto", Target: "Mixed"}
			if strings.Contains(content, "reverse_proxy") {
				site.Target = "Proxy"
			}
			sites = append(sites, site)
		}
	} else if ptype == ProxyApache {
		out, _ := e.client.Run("ls -1 /etc/apache2/sites-enabled/")
		lines := strings.Split(strings.TrimSpace(out), "\n")
		for _, l := range lines {
			if l == "" || strings.Contains(l, "No such file") {
				continue
			}
			confPath := fmt.Sprintf("/etc/apache2/sites-enabled/%s", l)
			content, _ := e.client.Run(fmt.Sprintf("cat %q", confPath))
			site := Site{Name: l, ConfPath: confPath, SSLStatus: "Inactive", Target: "Static"}
			if strings.Contains(content, "SSLEngine on") || strings.Contains(content, "443") {
				site.SSLStatus = "Active"
			}
			if strings.Contains(content, "ProxyPass") {
				site.Target = "Proxy"
			}
			sites = append(sites, site)
		}
	}
	return sites, nil
}

func (e *Engine) ReadConfig(path string) (string, error) {
	return e.client.Run(fmt.Sprintf("cat %q", path))
}

func (e *Engine) WriteConfig(path, content string) error {
	escaped := strings.ReplaceAll(content, "'", "'\"'\"'")
	_, err := e.client.Run(fmt.Sprintf("echo '%s' > %q", escaped, path))
	return err
}

func (e *Engine) ValidateSyntax(ptype ProxyType) (string, error) {
	switch ptype {
	case ProxyNginx:
		return e.client.Run("nginx -t 2>&1")
	case ProxyCaddy:
		return e.client.Run("caddy validate --config /etc/caddy/Caddyfile 2>&1")
	case ProxyApache:
		return e.client.Run("apache2ctl configtest 2>&1")
	}
	return "", fmt.Errorf("unknown proxy type")
}

func (e *Engine) ReloadRestart(ptype ProxyType) (string, error) {
	switch ptype {
	case ProxyNginx:
		return e.client.Run("systemctl reload nginx || systemctl restart nginx")
	case ProxyCaddy:
		return e.client.Run("systemctl reload caddy || systemctl restart caddy")
	case ProxyApache:
		return e.client.Run("systemctl reload apache2 || systemctl restart apache2")
	}
	return "", fmt.Errorf("unknown proxy type")
}
