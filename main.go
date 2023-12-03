package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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

const targetKey = "docker-pihole-customdns.domain"

type Action = string

const (
	CreateAction Action = "create"
	RemoveAction Action = "remove"
)

type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ExistingDNSResponse struct {
	Data [][]string `json:"data"`
}

var hostIP string
var authCode string
var pihole_url string
var authCode2 string
var pihole_url2 string

func main() {

	loadArguments()

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	testPiholeConnection()
	reconcileExistingDNS(cli)
	watchContainers(ctx, cli)
}

func loadArguments() {
	flag.StringVar(&hostIP, "hostip", "", "Docker host IP address")
	flag.StringVar(&authCode, "apitoken", "", "Pi-hole API token")
	flag.StringVar(&pihole_url, "piholeurl", "", "Pi-hole URL (http://pi.hole)")
	flag.StringVar(&authCode2, "apitoken2", "", "Second Pi-hole API token")
	flag.StringVar(&pihole_url2, "piholeurl2", "", "Second Pi-hole URL (http://pi.hole)")
	flag.Parse()

	if hostIP == "" {
		hostIP = os.Getenv("DPC_DOCKER_HOST_IP")
	}

	if authCode == "" {
		authCode = os.Getenv("DPC_PIHOLE_API_TOKEN")
	}

	if pihole_url == "" {
		pihole_url = os.Getenv("DPC_PIHOLE_URL")
	}

	if authCode2 == "" {
		authCode2 = os.Getenv("DPC_PIHOLE_API_TOKEN_2")
	}

	if pihole_url2 == "" {
		pihole_url2 = os.Getenv("DPC_PIHOLE_URL_2")
	}

	if hostIP == "" {
		log.Fatal("Docker host IP is not provided. Set it using the -hostip flag or DPC_DOCKER_HOST_IP environment variable.")
	}

	if authCode == "" {
		log.Fatal("Pi-hole API token is not provided. Set it using the -apitoken flag or DPC_PIHOLE_API_TOKEN environment variable.")
	}

	if pihole_url == "" {
		log.Fatal("Pi-hole URL is not provided. Set it using the -piholeurl flag or DPC_PIHOLE_URL environment variable.")
	}

	pihole_url += "/admin/api.php"

	if pihole_url2 != "" {
		pihole_url2 += "/admin/api.php"
		if authCode2 == "" {
			log.Fatal("Pi-hole API token is not provided. Set it using the -apitoken2 flag or DPC_PIHOLE_API_TOKEN_2 environment variable.")
		}
	}
}

func testPiholeConnection() {
	pihole_urls := []string{pihole_url}
	if pihole_url2 != "" { pihole_urls = append(pihole_urls, pihole_url2) }
	for _, url := range pihole_urls {
		testURL := url
		testURL += "?summaryRaw&auth="
		testURL += authCode
		resp, err := http.Get(testURL)
		if err != nil {
			log.Fatalf("Error connecting to Pi-hole %s: %v", url, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			bodyString := string(bodyBytes)
			if bodyString == "[]" {
				log.Fatalf("Error connecting to Pi-hole %s. Check API token.", url)
			} else {
				log.Printf("Connected to Pi-hole %s successfully", url)
			}
		} else {
			log.Fatalf("Error connecting to Pi-hole %s. Check API token.", url)
		}
	}
}

func reconcileExistingDNS(cli *client.Client){
	// Fetch existing DNS entries from Pi-hole
	existingDNS, err := getExistingDNS(pihole_url)
	if err != nil {
		log.Fatalf("Error fetching existing DNS entries from %s: %v", pihole_url, err)
	}
	// Check existing containers for the target key and create DNS records if needed
	checkExistingContainers(pihole_url, cli, existingDNS)

	if pihole_url2 != "" {
		// Fetch existing DNS entries from Pi-hole2
		existingDNS, err := getExistingDNS(pihole_url2)
		if err != nil {
			log.Fatalf("Error fetching existing DNS entries from %s: %v", pihole_url2, err)
		}
		// Check existing containers for the target key and create DNS records if needed
		checkExistingContainers(pihole_url2, cli, existingDNS)
	}
}

func getExistingDNS(pihole_url string) ([][]string, error) {
	// Make the API request to get existing DNS entries
	apiURL := pihole_url + "?customdns"
	apiURL += "&auth=" + authCode
	apiURL += "&action=get"
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch existing DNS entries. Status code: %d", resp.StatusCode)
	}

	// Decode the JSON response
	var existingDNSResponse ExistingDNSResponse
	if err := json.NewDecoder(resp.Body).Decode(&existingDNSResponse); err != nil {
		return nil, err
	}

	return existingDNSResponse.Data, nil
}

func checkExistingContainers(pihole_url string, cli *client.Client, existingDNS [][]string) {
	// Fetch all existing containers
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("Error fetching existing containers: %v", err)
	}

	// Iterate over containers and check for the target key
	for _, container := range containers {
		labelValue, found := container.Labels[targetKey]
		if found && isDNSMissing(labelValue, existingDNS) {
			// Strip off the "/" prefix from the container name
			containerName := strings.TrimPrefix(container.Names[0], "/")
			createDNS(pihole_url, containerName, labelValue, hostIP)
		}
	}
}

func isDNSMissing(labelValue string, existingDNS [][]string) bool {
	// Check if the DNS entry already exists
	for _, existing := range existingDNS {
		if len(existing) == 2 && existing[0] == labelValue && existing[1] == hostIP {
			return false
		}
	}
	return true
}

func watchContainers(ctx context.Context, cli *client.Client) {
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
					createDNS(pihole_url, event.Actor.Attributes["name"], label, hostIP)
					if pihole_url2 != "" {
						createDNS(pihole_url2, event.Actor.Attributes["name"], label, hostIP)
					}
				} else if action == RemoveAction {
					removeDNS(pihole_url, event.Actor.Attributes["name"], label, hostIP)
					if pihole_url2 != "" {
						removeDNS(pihole_url2, event.Actor.Attributes["name"], label, hostIP)
					}
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

func createDNS(pihole_url string, containerName string, domainName string, ipAddress string) {

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
		log.Printf("API for %s add request successful for container %s - %s", pihole_url, containerName, domainName)
	} else {
		log.Printf("API for %s add request failed for container %s - %s: %s", pihole_url, containerName, domainName, apiResponse.Message)
	}
}

func removeDNS(pihole_url string, containerName string, domainName string, ipAddress string) {
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
		log.Printf("API for %s delete request successful for container %s - %s", pihole_url, containerName, domainName)
	} else {
		log.Printf("API for %s delete request failed for container %s - %s: %s", pihole_url, containerName, domainName, apiResponse.Message)
	}
}
