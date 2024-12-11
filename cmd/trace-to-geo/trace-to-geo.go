package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ipinfo/go/v2/ipinfo"
	"github.com/ipinfo/go/v2/ipinfo/cache"
)

type Hop struct {
	Index int
	IP    net.IP
}

type Choice struct {
	Name        string
	ID          int
	Description string
}

func main() {
	// init
	choices := []Choice{
		{ID: 1, Description: "enter a new IP(s) or traceroute"},
		{ID: 2, Description: "Show full geo location data for each IP"},
		{ID: 3, Description: "Show initial traceroute with country appended"},
		{ID: 9, Description: "Exit program"},
	}
	var usrChoice int
	var usrInput []string
	var results ipinfo.BatchCore
	var ipList []Hop

	// start UI
	fmt.Print(`
---------------------------------------------------------------
Trace to Geo - based on ipinfo.io
				
First enter an ipinfo.io API token, then you can enter an IP, 
list of IPs or a traceroute. Then the program will output 
the geolocation data for each IP.
---------------------------------------------------------------
`)

	// get ipinfo.io API token
	token := getToken()

	// user interaction loop. usrChoice set to 1 for initially entering IPs
	usrChoice = 1
	for {
		switch usrChoice {
		case 1:
			// clear persistent variables
			results = nil
			usrInput = nil
			ipList = nil

			// Take and parse new input
			fmt.Println("Enter the IP(s), press Enter and then Ctrl+D or Ctrl+Z (depending on OS).")
			usrInput = readUserInput()
			if len(usrInput) > 0 {
				ipList = parseIPs(usrInput)
				results = queryIPs(ipList, token)
			} else {
				fmt.Println("No input detected, please try again")
			}

			// Select display method
			displayChoices(choices)
			usrChoice, _ = strconv.Atoi(readUserInputSingle())
		case 2:
			for _, hop := range ipList {
				currentHop := hop.IP.String()
				if info := results[currentHop]; info != nil {
					fmt.Println("---------------------------------------------------------------")
					fmt.Printf("Hop: %v - IP: %s\n", hop.Index, hop.IP)
					fmt.Println("---------------------------------------------------------------")
					fmt.Printf("Hostname: %s \n", info.Hostname)
					fmt.Printf("Anycast: %v \n", info.Anycast)
					fmt.Printf("City: %s \n", info.City)
					fmt.Printf("Region: %s \n", info.Region)
					fmt.Printf("Country: %s \n", info.Country)
					fmt.Printf("Location: %s \n", info.Location)
					fmt.Printf("Organization: %s \n", info.Org)
					fmt.Printf("Postal: %s \n", info.Postal)
					fmt.Printf("Timezone: %s \n", info.Timezone)
					fmt.Printf("\n\n")
				} else {
					fmt.Println("---------------------------------------------------------------")
					fmt.Printf("Hop: %v - IP: %s\n", hop.Index, hop.IP)
					fmt.Println("---------------------------------------------------------------")
					fmt.Printf("Hostname: %s \n", "Private IP")
					fmt.Printf("Anycast: %v \n", "Private IP")
					fmt.Printf("City: %s \n", "Private IP")
					fmt.Printf("Region: %s \n", "Private IP")
					fmt.Printf("Country: %s \n", "Private IP")
					fmt.Printf("Location: %s \n", "Private IP")
					fmt.Printf("Organization: %s \n", "Private IP")
					fmt.Printf("Postal: %s \n", "Private IP")
					fmt.Printf("Timezone: %s \n", "Private IP")
					fmt.Printf("\n\n")
				}
			}

			displayChoices(choices)
			usrChoice, _ = strconv.Atoi(readUserInputSingle())
		case 3:
			// find the longest trace line
			longestLine := findLongestLine(usrInput)

			// Find hop indexes and print lines
			reIndex := regexp.MustCompile(`^\s*\d* `)
			for _, l := range usrInput {
				// Find hop index to determine where to print IP details.
				hopIndex := strings.TrimSpace(reIndex.FindString(l))
				if len(hopIndex) > 0 {
					i, _ := strconv.Atoi(hopIndex)

					// find IP of current hop
					for _, hop := range ipList {
						if hop.Index == i {
							spaceDiff := strings.Repeat(" ", longestLine-len(l))
							currentHop := hop.IP.String()
							if info := results[currentHop]; info != nil {
								fmt.Printf("%s    %s# %s - %s\n", l, spaceDiff, info.City, info.CountryName)
							} else {
								fmt.Printf("%s    %s# %s - %s\n", l, spaceDiff, "Private IP", "Local")
							}
						}
					}

				} else {
					fmt.Printf("%s\n", l)
				}
			}

			displayChoices(choices)
			usrChoice, _ = strconv.Atoi(readUserInputSingle())
		case 9:
			fmt.Println("Good bye!")
			os.Exit(0)
		default:
			displayChoices(choices)
			usrChoice, _ = strconv.Atoi(readUserInputSingle())
		}
	}
}

func getToken() string {
	// Check if token flag is set
	tokenFlag := flag.String("t", "", "ipinfo.io API token")
	flag.Parse()

	if *tokenFlag != "" {
		return *tokenFlag
	}

	// If token flag not set check if environment variable is set
	if token := os.Getenv("IPINFO_TOKEN"); token != "" {
		return token
	}

	// if neither flag nor env is set, require user input
	fmt.Println("Enter ipinfo.io token:")
	var hasToken bool
	var token string
	for !hasToken {
		token = readUserInputSingle()
		if len(token) == 0 {
			fmt.Println("No token entered, try again..")
			continue
		} else if len(token) != 14 {
			fmt.Println("Invalid token length, try again..")
			continue
		}
		hasToken = true
	}
	return token
}

func displayChoices(choices []Choice) {
	fmt.Printf("\nSelect display option\n")
	for _, c := range choices {
		fmt.Printf("Enter \"%v\" for: %s\n", c.ID, c.Description)
	}
}

func queryIPs(hops []Hop, token string) ipinfo.BatchCore {
	// Create ipinfo client
	client := ipinfo.NewClient(
		nil,
		ipinfo.NewCache(cache.NewInMemory().WithExpiration(5*time.Minute)),
		token,
	)

	// create list of IPs to query, disregard private IPs
	var queryList []net.IP
	for _, hop := range hops {
		ip := hop.IP
		if !net.IP.IsPrivate(ip) {
			queryList = append(queryList, ip)
		}
	}

	// Query IPs
	result, err := client.GetIPInfoBatch(queryList, ipinfo.BatchReqOpts{})
	if err != nil {
		log.Fatal("Failed to query IPs", err)
	}

	return result
}

func parseIPs(usrInput []string) []Hop {
	// create a map for the hops, this way we can get the hop index later by "querying" the IP
	var hops []Hop
	var hopIndex int
	reIndex := regexp.MustCompile(`^\s*\d* `)

	for i, l := range usrInput {
		// first find the hop index in case of a traceroute
		if match := reIndex.FindStringSubmatch(l); match != nil {
			hopIndex, _ = strconv.Atoi(strings.TrimSpace(match[0]))
		} else {
			hopIndex = i + 1
		}

		// Divide line into parts for each string, and check if valid IP
		// If valid add it and the hop index to the list.
		parts := strings.Fields(l)
		for _, part := range parts {
			ip := net.ParseIP(part)
			if ip != nil {
				hop := Hop{Index: hopIndex, IP: ip}
				hops = append(hops, hop)
			}
		}
	}
	return hops
}

func findLongestLine(lines []string) int {
	longestLine := 0

	for _, l := range lines {
		lineLength := len(l)
		if lineLength > longestLine {
			longestLine = lineLength
		}
	}

	return longestLine
}

func readUserInput() []string {
	s := bufio.NewScanner(os.Stdin)

	var lines []string
	for {
		if !s.Scan() {
			break
		}
		lines = append(lines, s.Text())
	}

	err := s.Err()
	if err != nil {
		log.Fatal(err)
	}

	return lines
}

func readUserInputSingle() string {
	s := bufio.NewScanner(os.Stdin)
	s.Scan()
	ln := s.Text()
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}
	return ln
}
