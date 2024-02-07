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

type Responses struct {
	Website    string `json:"website"`
	StatusCode int    `json:"status"`
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

var sites = map[string]string{
	"Echecs France Results API": os.Getenv("ECHECS_FRANCE_RESULTS_API"),
	"Chess PDF API":             os.Getenv("CHESS_PDF_API"),
}

func getHealth(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}(resp.Body)
	return resp, nil
}

func checkHealthOfSites() ([]Responses, error) {
	var responses []Responses

	for website, url := range sites {
		response, err := getHealth(url)
		if err != nil {
			return nil, err
		}
		responses = append(responses, Responses{website, response.StatusCode})
	}

	return responses, nil
}

func createEmbed(responses []Responses, now time.Time) DiscordEmbed {
	var fields []EmbedField
	color := 0x00ff00 // default color is green for all status 200

	for _, response := range responses {
		fields = append(fields, EmbedField{
			Name:   response.Website,
			Value:  fmt.Sprintf("%d", response.StatusCode),
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
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}

	discordWebhook := os.Getenv("DISCORD_HEALTHCHECK_WEBHOOK")

	responses, err := checkHealthOfSites()
	if err != nil {
		log.Fatalf("Failed to check health of sites: %v", err)
	}

	now := time.Now()
	discordEmbed := createEmbed(responses, now)

	jsonData, err := json.Marshal(discordEmbed)
	if err != nil {
		log.Fatalf("Failed to marshal discord embed: %v", err)
	}

	resp, err := http.Post(discordWebhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to post to Discord: %v", err)
	}
	log.Println(resp.Status)
}
