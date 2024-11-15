package main

import (
	"bufio"
	"encoding/json"
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

var exitString string = "done"

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
	ID          int
	Description string
}

func main() {
	// init
	choices := []Choice{
		{ID: 0, Description: "Enter a new IP or traceroute"},
		{ID: 1, Description: "Show full IP info for every hop"},
		{ID: 2, Description: "Keep the same traceroute input, prepend Location info only"},
		{ID: 9, Description: "Exit program"},
	}
	var usrChoice int
	var token string
	var usrInput []string
	var results map[int]IPInfoResp
	var ipList map[int]string

	// start UI
	fmt.Print(`
---------------------------------------------------------------
Welcome to Trace to Geo
				
Please enter an IP or a traceroute (or anything containing an IP really)
and the program will output the geolocation data for each IP.
---------------------------------------------------------------
`)

	fmt.Println("Enter ipinfo.io token:")
	hasToken := false
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

	// user interaction loop
	for {
		switch usrChoice {
		case 9:
			fmt.Println("Good bye!")
			os.Exit(0)
		case 1:
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
					fmt.Printf("---------------------------------------------------------------\n\n")
				}
			}

			displayChoices(choices)
			usrChoice, _ = strconv.Atoi(readUserInputSingle())
		case 2:
			reIndex := regexp.MustCompile(`^\s*\d* `)

			for _, l := range usrInput {
				hopIndex := strings.TrimSpace(reIndex.FindString(l))

				if len(hopIndex) > 0 {
					i, _ := strconv.Atoi(hopIndex)
					fmt.Printf("%s        # %s - %s\n", l, results[i].City, results[i].Country)
				} else {
					fmt.Printf("%s\n", l)
				}
			}

			displayChoices(choices)
			usrChoice, _ = strconv.Atoi(readUserInputSingle())
		default:
			// clear persistent variables
			results = nil
			usrInput = nil
			ipList = nil

			// Take and parse new input
			fmt.Println("Enter the IP(s) and press and enter ", exitString, " in the last line.")
			usrInput = readUserInput()
			if len(usrInput) > 0 {
				ipList = parseIPs(usrInput)
				results = queryIPs(ipList, token)
			} else {
				fmt.Println("No input detected, please try again")
			}

			// Select display method
			fmt.Println("Select display option")
			displayChoices(choices)
			usrChoice, _ = strconv.Atoi(readUserInputSingle())
		}
	}
}

func displayChoices(choices []Choice) {
	for _, c := range choices {
		fmt.Println("Enter \"", c.ID, "\" for: ", c.Description)
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

func readUserInput() []string {
	s := bufio.NewScanner(os.Stdin)

	var lines []string
	for {
		s.Scan()
		l := s.Text()
		if l == exitString {
			break
		}
		lines = append(lines, l)
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
