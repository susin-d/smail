package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Settings struct {
	MySQLHost      string
	MySQLDatabase  string
	MySQLUser      string
	MySQLPassword  string
	JWTSecret      string
	JWTAlgorithm   string
	JWTExpireMins  int
	RedisURL       string
	SMTPHost       string
	SMTPPort       int
	IMAPHost       string
	IMAPPort       int
	CorsOriginList []string
	PrimaryDomain  string
	IsDev          bool
}

func GetSettings() *Settings {
	// Attempt to load .env, ignore if not found (for Docker)
	_ = godotenv.Load("../../.env")

	isDev := strings.ToLower(os.Getenv("MAAS_DEV")) == "1" ||
		strings.ToLower(os.Getenv("MAAS_DEV")) == "true" ||
		strings.ToLower(os.Getenv("MAAS_DEV")) == "yes"

	jwtExpire, err := strconv.Atoi(os.Getenv("JWT_EXPIRE_MINUTES"))
	if err != nil {
		jwtExpire = 1440
	}

	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		smtpPort = 587
	}

	imapPort, err := strconv.Atoi(os.Getenv("IMAP_PORT"))
	if err != nil {
		imapPort = 993
	}

	corsHeaders := os.Getenv("CORS_ORIGINS")
	var corsList []string
	if corsHeaders != "" {
		corsList = strings.Split(corsHeaders, ",")
	} else {
		corsList = []string{"*"}
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if !isDev && (strings.Contains(strings.ToLower(jwtSecret), "changeme") || len(jwtSecret) < 32) {
		log.Fatal("FATAL: JWT_SECRET must be at least 32 characters and not contain 'changeme'. Set a strong secret in your .env file.")
	}
	
	jwtAlg := os.Getenv("JWT_ALGORITHM")
	if jwtAlg == "" {
		jwtAlg = "HS256"
	}

	primaryDomain := os.Getenv("PRIMARY_DOMAIN")
	if primaryDomain == "" {
		primaryDomain = "example.com"
	}

	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		redisUrl = "redis://localhost:6379/0"
	}

	return &Settings{
		MySQLHost:      os.Getenv("MYSQL_HOST"),
		MySQLDatabase:  os.Getenv("MYSQL_DATABASE"),
		MySQLUser:      os.Getenv("MYSQL_USER"),
		MySQLPassword:  os.Getenv("MYSQL_PASSWORD"),
		JWTSecret:      jwtSecret,
		JWTAlgorithm:   jwtAlg,
		JWTExpireMins:  jwtExpire,
		RedisURL:       redisUrl,
		SMTPHost:       os.Getenv("SMTP_HOST"),
		SMTPPort:       smtpPort,
		IMAPHost:       os.Getenv("IMAP_HOST"),
		IMAPPort:       imapPort,
		CorsOriginList: corsList,
		PrimaryDomain:  primaryDomain,
		IsDev:          isDev,
	}
}
