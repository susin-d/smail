package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"api-go/internal/database"
	"api-go/internal/models"
)

type DomainHandler struct{}

type DnsRecord struct {
	RecordType string `json:"record_type"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	Priority   *int   `json:"priority,omitempty"`
}

type DomainDnsResponse struct {
	Domain  string      `json:"domain"`
	Records []DnsRecord `json:"records"`
}

type DomainResponse struct {
	ID            uint   `json:"id"`
	Domain        string `json:"domain"`
	IsVerified    bool   `json:"is_verified"`
	DKIMPublicKey string `json:"dkim_public_key,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type CreateDomainRequest struct {
	Domain string `json:"domain" binding:"required"`
}

func generateDNSRecords(domain string, dkimSelector string) []DnsRecord {
	if dkimSelector == "" {
		dkimSelector = "mail"
	}
	priority := 10
	return []DnsRecord{
		{RecordType: "MX", Name: domain, Value: fmt.Sprintf("mail.%s", domain), Priority: &priority},
		{RecordType: "A", Name: fmt.Sprintf("mail.%s", domain), Value: "YOUR_VPS_IP"},
		{RecordType: "TXT", Name: domain, Value: "v=spf1 mx a ~all"},
		{RecordType: "TXT", Name: fmt.Sprintf("%s._domainkey.%s", dkimSelector, domain), Value: "v=DKIM1; k=rsa; p=YOUR_DKIM_PUBLIC_KEY"},
		{RecordType: "TXT", Name: fmt.Sprintf("_dmarc.%s", domain), Value: fmt.Sprintf("v=DMARC1; p=quarantine; rua=mailto:postmaster@%s; ruf=mailto:postmaster@%s; fo=1", domain, domain)},
	}
}

func (h *DomainHandler) ListDomains(c *gin.Context) {
	var domains []models.Domain
	database.DB.Order("created_at desc").Find(&domains)

	response := make([]DomainResponse, len(domains))
	for i, d := range domains {
		response[i] = DomainResponse{
			ID:            d.ID,
			Domain:        d.Domain,
			IsVerified:    d.IsVerified,
			DKIMPublicKey: d.DKIMPublicKey,
			CreatedAt:     d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	c.JSON(http.StatusOK, response)
}

func (h *DomainHandler) CreateDomain(c *gin.Context) {
	var req CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "Invalid input"})
		return
	}

	var existing models.Domain
	if err := database.DB.Where("domain = ?", req.Domain).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"detail": "Domain already registered"})
		return
	}

	domain := models.Domain{Domain: req.Domain}
	if err := database.DB.Create(&domain).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to create domain"})
		return
	}

	records := generateDNSRecords(req.Domain, "mail")
	c.JSON(http.StatusCreated, DomainDnsResponse{
		Domain:  req.Domain,
		Records: records,
	})
}

func (h *DomainHandler) GetDomainDNS(c *gin.Context) {
	domainID := c.Param("domain_id")
	var domain models.Domain
	if err := database.DB.First(&domain, domainID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Domain not found"})
		return
	}

	records := generateDNSRecords(domain.Domain, "mail")
	c.JSON(http.StatusOK, DomainDnsResponse{
		Domain:  domain.Domain,
		Records: records,
	})
}

func (h *DomainHandler) DeleteDomain(c *gin.Context) {
	domainID := c.Param("domain_id")
	var domain models.Domain
	if err := database.DB.First(&domain, domainID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Domain not found"})
		return
	}

	database.DB.Delete(&domain)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Domain '%s' deleted", domain.Domain)})
}
