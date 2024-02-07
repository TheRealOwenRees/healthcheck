package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type CertificateDetails struct {
	Issuer     string
	Domain     string
	ValidFrom  time.Time
	ValidUntil time.Time
}

type Responses struct {
	Website    string `json:"website"`
	StatusCode int    `json:"status"`
	CertificateDetails
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type Embed struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Color       int          `json:"color"`
	Fields      []EmbedField `json:"fields"`
}

type DiscordEmbed struct {
	Embeds []Embed `json:"embeds"`
}

const DateFormat = "02/01/2006 15:04:05"

// getHealth makes a GET request to the given URL and returns the response and certificate details
func getHealth(url string) (*http.Response, CertificateDetails, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, CertificateDetails{}, err
	}

	var certDetails CertificateDetails
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		certDetails = CertificateDetails{
			Issuer:     cert.Issuer.Organization[0],
			Domain:     cert.Subject.CommonName,
			ValidFrom:  cert.NotBefore,
			ValidUntil: cert.NotAfter,
		}
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}(resp.Body)
	return resp, certDetails, nil
}

// checkHealthOfSites iterates through the sites detailed in main() and calls getHealth for each one
func checkHealthOfSites(sites map[string]string) ([]Responses, error) {
	var responses []Responses

	for website, url := range sites {
		response, certDetails, err := getHealth(url)
		if err != nil {
			return nil, err
		}
		responses = append(responses, Responses{website, response.StatusCode, certDetails})
	}

	return responses, nil
}

// createEmbed creates a Discord embed with the health check results, ready for sending via the Discord webhook
func createEmbed(responses []Responses, now time.Time) DiscordEmbed {
	var fields []EmbedField
	color := 0x00ff00 // default color is green for all status 200

	for _, response := range responses {
		fields = append(fields, EmbedField{
			Name: response.Website,
			Value: fmt.Sprintf("Status: %d\n\n*SSL/TLS Status*\n-----------------\nIssuer: %s\nDomain: %s\nValid From: %s\nValid Until: %s",
				response.StatusCode,
				response.Issuer,
				response.Domain,
				response.ValidFrom.Format(DateFormat),
				response.ValidUntil.Format(DateFormat),
			),
			Inline: true,
		})

		if response.StatusCode != 200 {
			color = 0xff0000 // if any status is not 200, change color to red
		}
	}

	dateTimeTitle := fmt.Sprintf("Health Check for %s", now.Format(DateFormat))

	embed := Embed{
		Title:       dateTimeTitle,
		Description: "Website Health Check",
		Color:       color,
		Fields:      fields,
	}

	return DiscordEmbed{
		Embeds: []Embed{embed},
	}
}

func main() {
	// load '.env' file if it exists (dev environment) or use OS environment variables (prod environment)
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}

	// change this to your own Discord webhook
	discordWebhook := os.Getenv("DISCORD_HEALTHCHECK_WEBHOOK")

	// change this to your own sites, set as environment variables
	sites := map[string]string{
		"Echecs France Results API": os.Getenv("ECHECS_FRANCE_RESULTS_API"),
		"Chess PDF API":             os.Getenv("CHESS_PDF_API"),
	}

	// check health of sites and return responses including certificate details
	responses, err := checkHealthOfSites(sites)
	if err != nil {
		log.Fatalf("Failed to check health of sites: %v", err)
	}

	// create Discord embed and send to Discord webhook
	now := time.Now()
	discordEmbed := createEmbed(responses, now)
	jsonData, err := json.Marshal(discordEmbed)
	if err != nil {
		log.Fatalf("Failed to marshal discord embed: %v", err)
	}

	// send to Discord
	resp, err := http.Post(discordWebhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to post to Discord: %v", err)
	}
	log.Println(resp.Status)
}
