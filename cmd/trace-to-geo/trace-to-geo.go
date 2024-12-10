package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type IPInfoResp struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	Anycast  bool   `json:"anycast"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
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
	var results map[int]IPInfoResp
	var ipList map[int]string

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
			keys := make([]int, 0, len(results))
			for k := range results {
				keys = append(keys, k)
			}
			sort.Ints(keys)

			for _, k := range keys {
				if r, exist := results[k]; exist {
					fmt.Println("---------------------------------------------------------------")
					fmt.Println("Hop", k, "IP:", r.IP)
					fmt.Println("---------------------------------------------------------------")
					fmt.Printf("Hostname: %s \n", r.Hostname)
					fmt.Printf("Anycast: %v \n", r.Anycast)
					fmt.Printf("City: %s \n", r.City)
					fmt.Printf("Region: %s \n", r.Region)
					fmt.Printf("Country: %s \n", r.Country)
					fmt.Printf("Location: %s \n", r.Loc)
					fmt.Printf("Organization: %s \n", r.Org)
					fmt.Printf("Postal: %s \n", r.Postal)
					fmt.Printf("Timezone: %s \n", r.Timezone)
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
				hopIndex := strings.TrimSpace(reIndex.FindString(l))

				if len(hopIndex) > 0 {
					i, _ := strconv.Atoi(hopIndex)
					spaceDiff := strings.Repeat(" ", longestLine-len(l))
					fmt.Printf("%s    %s# %s - %s\n", l, spaceDiff, results[i].City, results[i].Country)
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

func queryIPs(ipList map[int]string, token string) map[int]IPInfoResp {
	results := make(map[int]IPInfoResp)
	baseURL := "https://ipinfo.io"
	reRFC1918 := regexp.MustCompile(`^(10\.(?:\d{1,3}\.){2}\d{1,3})$|^(172\.(?:1[6-9]|2[0-9]|3[0-1])\.\d{1,3}\.\d{1,3})$|^(192\.168\.\d{1,3}\.\d{1,3})$`)

	for i, ip := range ipList {
		if match := reRFC1918.FindStringSubmatch(ip); match != nil {
			results[i] = IPInfoResp{IP: ip, City: "Local"}
			continue
		} else {
			u, err := url.Parse(baseURL)
			if err != nil {
				fmt.Println("Error parsing URL: ", err)
				return nil
			}

			u.Path += "/" + ip
			q := u.Query()
			q.Add("token", token)
			u.RawQuery = q.Encode()

			resp, err := http.Get(u.String())
			if err != nil {
				log.Println(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Println(err)
				}

				var result IPInfoResp
				if err := json.Unmarshal(body, &result); err != nil {
					fmt.Println("Can not unmarshal JSON")
					break
				}
				results[i] = result
			} else {
				fmt.Println("IP query failed, http status: ", resp.StatusCode, " - ", resp.Status)
			}
		}
	}
	return results
}

func parseIPs(usrInput []string) map[int]string {
	ipList := make(map[int]string)

	reIP := regexp.MustCompile(`(?:25[0-5]|2[0-4]\d|1\d{2}|\d{1,2})(?:\.(?:25[0-5]|2[0-4]\d|1\d{2}|\d{1,2})){3}`)
	reIndex := regexp.MustCompile(`^\s*\d* `)

	for i, l := range usrInput {
		hopIndex := strings.TrimSpace(reIndex.FindString(l))
		ip := reIP.FindString(l)

		if (len(ip) > 0) && (len(hopIndex) > 0) {
			hop, _ := strconv.Atoi(hopIndex)
			ipList[hop] = ip
		} else if len(ip) > 0 {
			ipList[i+1] = ip
		}
	}
	return ipList
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
