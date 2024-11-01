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
	"strconv"
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

	// start UI
	fmt.Print(`
---------------------------------------------------------------
Welcome to Trace to Geo
				
Please enter an IP or a traceroute (or anything containing an IP really)
and the program will output the geolocation data for each IP.
---------------------------------------------------------------
`)
	var usrChoice int
	var results []IPInfoResp
	fmt.Println("Enter ipinfo.io token:")
	token := readUserInputSingle()

	// user interaction loop
	for {
		switch usrChoice {
		case 9:
			fmt.Println("Good bye!")
			os.Exit(0)
		case 1:
			for i, result := range results {
				fmt.Println("---------------------------------------------------------------")
				fmt.Println("Hop", i+1, "IP:", result.IP)
				fmt.Println("---------------------------------------------------------------")
				fmt.Printf("Hostname: %s \n", result.Hostname)
				fmt.Printf("Anycast: %v \n", result.Anycast)
				fmt.Printf("City: %s \n", result.City)
				fmt.Printf("Region: %s \n", result.Region)
				fmt.Printf("Country: %s \n", result.Country)
				fmt.Printf("Location: %s \n", result.Loc)
				fmt.Printf("Organization: %s \n", result.Org)
				fmt.Printf("Postal: %s \n", result.Postal)
				fmt.Printf("Timezone: %s \n", result.Timezone)
				fmt.Printf("---------------------------------------------------------------\n\n")
			}

			displayChoices(choices)
			usrChoice, _ = strconv.Atoi(readUserInputSingle())
		case 2:
			displayChoices(choices)
			usrChoice, _ = strconv.Atoi(readUserInputSingle())
		default:
			results = nil
			fmt.Println("Enter the IP(s) and press and enter ", exitString, " in the last line.")
			usrInput := readUserInput()
			ipList := parseIPs(usrInput)
			results = queryIPs(ipList, token)

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

func queryIPs(ipList []string, token string) []IPInfoResp {
	var results []IPInfoResp
	baseURL := "https://ipinfo.io"

	for _, ip := range ipList {

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

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
		}

		var result IPInfoResp
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Println("Can not unmarshal JSON")
			break
		}
		results = append(results, result)
	}
	return results
}

func parseIPs(usrInput []string) []string {
	var ipList []string

	reIP := regexp.MustCompile(`(?:25[0-5]|2[0-4]\d|1\d{2}|\d{1,2})(?:\.(?:25[0-5]|2[0-4]\d|1\d{2}|\d{1,2})){3}`)

	for _, l := range usrInput {
		ip := reIP.FindString(l)
		if len(ip) > 0 {
			ipList = append(ipList, ip)
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
