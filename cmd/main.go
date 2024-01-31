package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
const DiscordWebhook = "https://discord.com/api/webhooks/1202341979208294450/zSlgbQ4X4gJ3zXsKwRJ6lqxs7QtpN8dtJENd0Z_vP6c-shFCWa7xipGTPOgwvMQL505B"

var sites = map[string]string{
	"Echecs France Results API": "https://api.echecsfrance.com/api/v1/health",
	"Chess PDF API":             "https://api.chess-scribe.org/api/v1/health",
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

	resp, err := http.Post(DiscordWebhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to post to Discord: %v", err)
	}
	log.Println(resp.Status)
}
