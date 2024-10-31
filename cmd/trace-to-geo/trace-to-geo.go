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

func main() {
	// start UI
	fmt.Print(`
------------------------------------------------------------
Welcome to Trace to Geo
				
Please enter an IP or a traceroute (or anything containing an IP really)
and the program will output the geolocation data for each IP.
------------------------------------------------------------
`)
	fmt.Println("Enter ipinfo.io token:")
	token := readUserInputSingle()

	fmt.Println("Enter the IP(s) and press and enter ", exitString, " in the last line.")

	usrInput := readUserInput()
	ipList := parseIPs(usrInput)

	for _, ip := range ipList {
		baseURL := "https://ipinfo.io"

		u, err := url.Parse(baseURL)
		if err != nil {
			fmt.Println("Error parsing URL: ", err)
			return
		}

		u.Path += "/" + ip
		q := u.Query()
		q.Add("token", token)
		u.RawQuery = q.Encode()

		resp, err := http.Get(u.String())
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var result IPInfoResp
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Println("Can not unmarshal JSON")
		}
		fmt.Println(PrettyPrint(result.Country))
	}

	fmt.Println("Press enter to exit program")
	readUserInputSingle()
}

func PrettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
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
