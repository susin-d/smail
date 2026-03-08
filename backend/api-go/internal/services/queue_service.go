package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"api-go/internal/database"
)

type MailJob struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Sender     string  `json:"sender"`
	Recipient  string  `json:"recipient"`
	Subject    string  `json:"subject"`
	Body        string   `json:"body"`
	HTMLBody    *string  `json:"html_body"`
	Attachments []string `json:"attachments"`
	UserID      uint     `json:"user_id"`
	Retries     int      `json:"retries"`
	MaxRetries  int      `json:"max_retries"`
	CreatedAt   string   `json:"created_at"`
}

func EnqueueMailJob(sender, recipient, subject, body string, htmlBody *string, attachments []string, userID uint) string {
	jobID := fmt.Sprintf("mail:%d", time.Now().UnixNano())

	job := MailJob{
		ID:          jobID,
		Type:        "send_mail",
		Sender:      sender,
		Recipient:   recipient,
		Subject:     subject,
		Body:        body,
		HTMLBody:    htmlBody,
		Attachments: attachments,
		UserID:      userID,
		Retries:     0,
		MaxRetries:  3,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(job)
	if err != nil {
		log.Printf("Failed to marshal mail job: %v", err)
		return ""
	}

	ctx := context.Background()
	if err := database.RedisClient.LPush(ctx, "smail:mail_queue", data).Err(); err != nil {
		log.Printf("Failed to enqueue mail job to redis: %v", err)
	} else {
		log.Printf("Enqueued mail job: %s (%s -> %s)", jobID, sender, recipient)
	}

	return jobID
}

func DequeueMailJob() (*MailJob, error) {
	ctx := context.Background()
	
	// rpop is non-blocking. return nil if empty.
	result, err := database.RedisClient.RPop(ctx, "smail:mail_queue").Result()
	if err != nil {
		return nil, err // Returns redis.Nil if empty
	}

	var job MailJob
	if err := json.Unmarshal([]byte(result), &job); err != nil {
		return nil, err
	}

	return &job, nil
}

func EnqueueRetry(job *MailJob) {
	job.Retries++
	data, _ := json.Marshal(job)
	ctx := context.Background()

	if job.Retries <= job.MaxRetries {
		database.RedisClient.LPush(ctx, "smail:mail_queue", data)
		log.Printf("Re-enqueued job %s (retry %d)", job.ID, job.Retries)
	} else {
		database.RedisClient.LPush(ctx, "smail:dead_letter", data)
		log.Printf("Job %s exceeded max retries, moved to dead letter queue", job.ID)
	}
}
