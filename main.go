package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const targetKey = "uk.noyes.docker-pihole-ddns.domain"
type Action = string

const (
	CreateAction Action = "create"
	RemoveAction Action = "remove"
)

type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

var hostIP string
var authCode string
var pihole_url string

func main() {
	// Parse command-line arguments
	flag.StringVar(&hostIP, "hostip", "", "Docker host IP address")
	flag.StringVar(&authCode, "apitoken", "", "Pi-hole API token")
	flag.StringVar(&pihole_url, "piholeurl", "", "Pi-hole URL (http://pi.hole)")
	flag.Parse()

	// If hostIP is not provided via command line, check the environment variable
	if hostIP == "" {
		hostIP = os.Getenv("DPD_DOCKER_HOST_IP")
	}

	// If authCode is not provided via command line, check the environment variable
	if authCode == "" {
		authCode = os.Getenv("DPD_PIHOLE_API_TOKEN")
	}

	// If pihole_url is not provided via command line, check the environment variable
	if pihole_url == "" {
		pihole_url = os.Getenv("DPD_PIHOLE_URL")
	}

	// Validate that hostIP is provided
	if hostIP == "" {
		log.Fatal("Docker host IP is not provided. Set it using the -hostip flag or DPD_DOCKER_HOST_IP environment variable.")
	}

	// Validate that authCode is provided
	if authCode == "" {
		log.Fatal("Pi-hole API token is not provided. Set it using the -apitoken flag or DPD_PIHOLE_API_TOKEN environment variable.")
	}

	// Validate that pihole_url is provided
	if pihole_url == "" {
		log.Fatal("Pi-hole URL is not provided. Set it using the -piholeurl flag or DPD_PIHOLE_URL environment variable.")
	}

	pihole_url += "/admin/api.php"

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	testURL := pihole_url
	testURL += "?summaryRaw&auth="
	testURL += authCode
	resp, err := http.Get(testURL)
	if err != nil {
		log.Printf("Error connecting to Pi-hole: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		if bodyString == "[]" {
			log.Fatal("Error connecting to Pi-hole. Check API token.")
		} else {
			log.Print("Connected to Pi-hole successfully")
		}
	}

	options := types.EventsOptions{}
	options.Filters = filters.NewArgs()
	options.Filters.Add("type", events.ContainerEventType)
	options.Filters.Add("event", CreateAction)
	options.Filters.Add("event", RemoveAction)

	events, errs := cli.Events(ctx, options)

	for {
		select {
		case event := <-events:
			relevant, action, label := isRelevantEvent(event)
			if relevant {
				if action == CreateAction {
					createDNS(event.Actor.Attributes["name"], label, hostIP)
				} else if action == RemoveAction {
					removeDNS(event.Actor.Attributes["name"], label, hostIP)
				}
				
			}
		case err := <-errs:
			log.Fatalf("Error watching events: %v", err)
		}
	}
}

func isRelevantEvent(event events.Message) (bool, Action, string) {
	// Check if the container has the target key
	for key, value := range event.Actor.Attributes {
		if strings.ToLower(key) == targetKey {
			return true, event.Action, strings.ToLower(value)
		}
	}

	return false, "", ""
}

func createDNS(containerName string, domainName string, ipAddress string) {
	
	// Make the API request with the required parameters
	apiURL := pihole_url + "?customdns"
	apiURL += "&auth=" + authCode
	apiURL += "&action=add"
	apiURL += "&ip=" + ipAddress
	apiURL += "&domain=" + domainName

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("Error making API request: %v", err)
		return
	}
	defer resp.Body.Close()

	// Decode the JSON response
	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		log.Printf("Error decoding JSON response: %v", err)
		return
	}

	// Check the "success" attribute in the response
	if apiResponse.Success {
		log.Printf("API add request successful for container %s: %s", containerName, apiResponse.Message)
	} else {
		log.Printf("API add request failed for container %s: %s", containerName, apiResponse.Message)
	}
}

func removeDNS(containerName string, domainName string, ipAddress string) {
	// Make the API request with the required parameters
	apiURL := pihole_url + "?customdns"
	apiURL += "&auth=" + authCode
	apiURL += "&action=delete"
	apiURL += "&ip=" + ipAddress
	apiURL += "&domain=" + domainName

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("Error making API request: %v", err)
		return
	}
	defer resp.Body.Close()

	// Decode the JSON response
	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		log.Printf("Error decoding JSON response: %v", err)
		return
	}

	// Check the "success" attribute in the response
	if apiResponse.Success {
		log.Printf("API delete request successful for container %s: %s", containerName, apiResponse.Message)
	} else {
		log.Printf("API delete request failed for container %s: %s", containerName, apiResponse.Message)
	}
}
