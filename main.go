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
	"github.com/docker/docker/api/types/container"
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

var defaultTargetIP string
var defaultTargetDomain string
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
	flag.StringVar(&defaultTargetIP, "targetip", "", "Default target IP address for the Docker host")
	flag.StringVar(&defaultTargetDomain, "targetdomain", "", "Default target domain address for the Docker host")
	flag.StringVar(&authCode, "apitoken", "", "Pi-hole API token")
	flag.StringVar(&pihole_url, "piholeurl", "", "Pi-hole URL (e.g. http://pi.hole)")
	flag.StringVar(&authCode2, "apitoken2", "", "Second Pi-hole API token (Optional)")
	flag.StringVar(&pihole_url2, "piholeurl2", "", "Second Pi-hole URL (Optional e.g. http://pi.hole)")
	flag.Parse()

	if defaultTargetIP == "" {
		defaultTargetIP = os.Getenv("DPC_DEFAULT_TARGET_IP")
	}

	if defaultTargetDomain == "" {
		defaultTargetDomain = os.Getenv("DPC_DEFAULT_TARGET_DOMAIN")
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

	if defaultTargetIP == "" && defaultTargetDomain == "" {
		log.Fatal("Default Docker host target IP or target domain are not provided. Set either using the -targetip flag (DPC_DEFAULT_TARGET_IP) or -targetdomain (DPC_DEFAULT_TARGET_DOMAIN).")
	} else if defaultTargetIP != "" && defaultTargetDomain != "" {
		log.Fatal("Both default target IP and target domain are set. Only one default can be used.")
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
	if defaultTargetIP == "" {
		apiURL = pihole_url + "?customcname"
	}
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
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		log.Fatalf("Error fetching existing containers: %v", err)
	}

	// Iterate over containers and check for the target key
	for _, container := range containers {
		labelValue, found := container.Labels[targetKey]
		if found && isDNSMissing(labelValue, existingDNS) {
			// Strip off the "/" prefix from the container name
			containerName := strings.TrimPrefix(container.Names[0], "/")
			createDNSRecord(pihole_url, containerName, labelValue)
		}
	}
}

func isDNSMissing(labelValue string, existingDNS [][]string) bool {
	// Check if the DNS entry already exists
	searchRecord := defaultTargetIP
	if defaultTargetIP == "" {
		searchRecord = defaultTargetDomain
	}
	for _, existing := range existingDNS {
		if len(existing) == 2 && existing[0] == labelValue && existing[1] == searchRecord {
			return false
		}
	}
	return true
}

func watchContainers(ctx context.Context, cli *client.Client) {
	options := types.EventsOptions{}
	options.Filters = filters.NewArgs()
	options.Filters.Add("type", string(events.ContainerEventType))
	options.Filters.Add("event", CreateAction)
	options.Filters.Add("event", RemoveAction)

	events, errs := cli.Events(ctx, options)

	for {
		select {
		case event := <-events:
			relevant, action, label := isRelevantEvent(event)
			if relevant {
				if action == CreateAction {
					createDNSRecord(pihole_url, event.Actor.Attributes["name"], label)
					if pihole_url2 != "" {
						createDNSRecord(pihole_url2, event.Actor.Attributes["name"], label)
					}
				} else if action == RemoveAction {
					removeDNSRecord(pihole_url, event.Actor.Attributes["name"], label)
					if pihole_url2 != "" {
						removeDNSRecord(pihole_url2, event.Actor.Attributes["name"], label)
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
			return true, string(event.Action), strings.ToLower(value)
		}
	}

	return false, "", ""
}

func createDNSRecord(pihole_url string, containerName string, domainName string) {
	if defaultTargetIP != "" {
		createARecord(pihole_url, containerName, domainName, defaultTargetIP)
	} else {
		createCNAMERecord(pihole_url, containerName, domainName, defaultTargetDomain)
	}
}

func removeDNSRecord(pihole_url string, containerName string, domainName string) {
	if defaultTargetIP != "" {
		removeARecord(pihole_url, containerName, domainName, defaultTargetIP)
	} else {
		removeCNAMERecord(pihole_url, containerName, domainName, defaultTargetDomain)
	}
}

func createARecord(pihole_url string, containerName string, domainName string, ipAddress string) {

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

func removeARecord(pihole_url string, containerName string, domainName string, ipAddress string) {
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

func createCNAMERecord(pihole_url string, containerName string, domainName string, targetName string) {

	// Make the API request with the required parameters
	apiURL := pihole_url + "?customcname"
	apiURL += "&auth=" + authCode
	apiURL += "&action=add"
	apiURL += "&target=" + targetName
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

func removeCNAMERecord(pihole_url string, containerName string, domainName string, targetName string) {
	// Make the API request with the required parameters
	apiURL := pihole_url + "?customcname"
	apiURL += "&auth=" + authCode
	apiURL += "&action=delete"
	apiURL += "&target=" + targetName
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
