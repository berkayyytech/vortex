package deploy

import (
	"fmt"
	"io"
	"net/http"
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

// Deploy runs a deployment pipeline with git pull, build, restart, and health check.
// It returns a channel for live step-by-step output.
func (e *Engine) Deploy(appDir string, buildCmd string, restartCmd string, healthUrl string) (<-chan string, error) {
	outChan := make(chan string, 100)

	go func() {
		defer close(outChan)

		runStep := func(stepName, cmd string) error {
			outChan <- fmt.Sprintf("▶ [%s] Executing: %s", stepName, cmd)
			out, err := e.client.Run(cmd)
			if err != nil {
				outChan <- fmt.Sprintf("✖ [%s] Failed: %v\nOutput: %s", stepName, err, out)
				e.LogAudit(fmt.Sprintf("Pipeline failed at %s. Error: %v", stepName, err))
				return err
			}
			if strings.TrimSpace(out) != "" {
				outChan <- fmt.Sprintf("✔ [%s] Success.\n%s", stepName, out)
			} else {
				outChan <- fmt.Sprintf("✔ [%s] Success.", stepName)
			}
			return nil
		}

		e.LogAudit(fmt.Sprintf("Starting deployment pipeline in %s", appDir))
		outChan <- "▶ Starting Deployment Pipeline"

		// 1. Git Pull
		if err := runStep("Git Pull", fmt.Sprintf("cd %s && git pull", appDir)); err != nil {
			return
		}

		// 2. Build
		if buildCmd != "" {
			if err := runStep("Build", fmt.Sprintf("cd %s && %s", appDir, buildCmd)); err != nil {
				return
			}
		}

		// 3. Restart
		if restartCmd != "" {
			if err := runStep("Restart", fmt.Sprintf("cd %s && %s", appDir, restartCmd)); err != nil {
				return
			}
		}

		// 4. Health Check
		if healthUrl != "" {
			outChan <- fmt.Sprintf("▶ [Health] Checking URL: %s", healthUrl)
			// check with curl
			healthCmd := fmt.Sprintf("curl -s -o /dev/null -w \"%%{http_code}\" %s", healthUrl)
			out, err := e.client.Run(healthCmd)
			if err != nil {
				outChan <- fmt.Sprintf("✖ [Health] Failed to run curl: %v", err)
				e.LogAudit("Deployment failed health check (curl error).")
				return
			}
			out = strings.TrimSpace(out)
			// accept 200-299 status codes or 301/302 redirects
			if strings.HasPrefix(out, "2") || strings.HasPrefix(out, "3") {
				outChan <- fmt.Sprintf("✔ [Health] Status %s. Health check passed.", out)
			} else {
				outChan <- fmt.Sprintf("✖ [Health] Status %s. Health check FAILED.", out)
				e.LogAudit(fmt.Sprintf("Deployment failed health check with status %s.", out))
				return
			}
		}

		e.LogAudit(fmt.Sprintf("Deployment completed successfully in %s", appDir))
		outChan <- "✔ Deployment Pipeline Completed Successfully"
	}()

	return outChan, nil
}

// LogAudit logs an action to the deployment audit log on the server.
func (e *Engine) LogAudit(message string) {
	// Escape double quotes for shell execution
	escaped := strings.ReplaceAll(message, "\"", "\\\"")
	cmd := fmt.Sprintf(`echo "[$(date -u)] %s" >> /var/log/vortex-deploy-audit.log`, escaped)
	e.client.Run(cmd)
}

// StartWebhookListener starts a simple HTTP server on the given port to trigger the deployment pipeline.
func (e *Engine) StartWebhookListener(port string, route string, appDir string, buildCmd string, restartCmd string, healthUrl string) error {
	mux := http.NewServeMux()
	mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read body to verify webhook payload if needed (ignored for now)
		io.ReadAll(r.Body)
		r.Body.Close()
		
		e.LogAudit("Webhook received. Triggering deployment.")
		
		// Run deployment in background
		outChan, err := e.Deploy(appDir, buildCmd, restartCmd, healthUrl)
		if err != nil {
			http.Error(w, "Failed to start deployment", http.StatusInternalServerError)
			return
		}
		
		go func() {
			for out := range outChan {
				_ = out // drain the channel so it doesn't block
			}
		}()

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Deployment triggered successfully."))
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		// Start the server in the background
		e.LogAudit(fmt.Sprintf("Starting Webhook listener on port %s for route %s", port, route))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			e.LogAudit(fmt.Sprintf("Webhook listener failed: %v", err))
		}
	}()

	return nil
}
