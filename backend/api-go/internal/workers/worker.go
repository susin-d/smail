package workers

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"api-go/internal/config"
	"api-go/internal/services"
	"github.com/redis/go-redis/v9"
)

func SendEmail(cfg *config.Settings, sender, recipient, subject, body string, htmlBody *string, attachments []string) error {
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := c.Mail(sender); err != nil {
		return err
	}
	if err := c.Rcpt(recipient); err != nil {
		return err
	}

	w, err := c.Data()
	if err != nil {
		return err
	}
	defer w.Close()

	// Write Headers
	fmt.Fprintf(w, "To: %s\r\nFrom: %s\r\nSubject: %s\r\n", recipient, sender, subject)

	boundary := "my-boundary-779"
	if htmlBody != nil || len(attachments) > 0 {
		fmt.Fprintf(w, "MIME-version: 1.0;\r\nContent-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary)

		// Text Part
		fmt.Fprintf(w, "--%s\r\n", boundary)
		if htmlBody != nil && *htmlBody != "" {
			altBoundary := "alt-boundary-779"
			fmt.Fprintf(w, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altBoundary)

			fmt.Fprintf(w, "--%s\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s\r\n\r\n", altBoundary, body)
			fmt.Fprintf(w, "--%s\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n\r\n", altBoundary, *htmlBody)
			fmt.Fprintf(w, "--%s--\r\n", altBoundary)
		} else {
			fmt.Fprintf(w, "Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s\r\n\r\n", body)
		}

		// Attachments
		for _, att := range attachments {
			f, err := os.Open(att)
			if err != nil {
				log.Printf("Warning: Failed to open attachment %s: %v", att, err)
				continue
			}

			// Extract original filename (removing timestamp prefix `int64_name.ext`)
			baseName := filepath.Base(att)
			parts := strings.SplitN(baseName, "_", 2)
			if len(parts) == 2 {
				baseName = parts[1]
			}

			fmt.Fprintf(w, "--%s\r\n", boundary)
			fmt.Fprintf(w, "Content-Type: application/octet-stream; name=\"%s\"\r\n", baseName)
			fmt.Fprintf(w, "Content-Transfer-Encoding: base64\r\n")
			fmt.Fprintf(w, "Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", baseName)

			// Stream base64 directly to network
			b64w := base64.NewEncoder(base64.StdEncoding, w)
			io.Copy(b64w, f)
			b64w.Close()
			fmt.Fprintf(w, "\r\n\r\n")

			f.Close()
			os.Remove(att) // Cleanup temp file after reading
		}
		fmt.Fprintf(w, "--%s--\r\n", boundary)
	} else {
		fmt.Fprintf(w, "\r\n%s\r\n", body)
	}

	return nil
}

func WorkerLoop(cfg *config.Settings) {
	log.Println("MaaS Mail Worker started. Waiting for jobs...")

	for {
		job, err := services.DequeueMailJob()
		if err == redis.Nil || job == nil {
			time.Sleep(2 * time.Second)
			continue
		} else if err != nil {
			log.Printf("Worker redis error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		err = SendEmail(cfg, job.Sender, job.Recipient, job.Subject, job.Body, job.HTMLBody, job.Attachments)
		if err != nil {
			log.Printf("✗ Job %s failed: %v", job.ID, err)
			services.EnqueueRetry(job)
		} else {
			log.Printf("✓ Job %s delivered: %s -> %s", job.ID, job.Sender, job.Recipient)
		}
	}
}
