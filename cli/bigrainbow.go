package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"
	"time"
)

const (
	configDirName  = ".bigrainbow"
	configFileName = "config.json"

	hexEncoding    = "hex"
	base64Encoding = "base64"

	jsonMime = "application/json"

	normal = "\033[0m"
	black  = "\033[30m"
	red    = "\033[31m"
	green  = "\033[32m"
	orange = "\033[33m"
	blue   = "\033[34m"
	purple = "\033[35m"
	cyan   = "\033[36m"
	gray   = "\033[37m"
	bold   = "\033[1m"
	clear  = "\r\x1b[2K"
	upN    = "\033[%dA"
	downN  = "\033[%dB"

	// INFO - Informational prompt
	INFO = bold + cyan + "[*] " + normal
	// WARN - Warn prompt
	WARN = bold + red + "[!] " + normal
	// READ - Read from user prompt
	READ = bold + purple + "[?] " + normal
	// WOOT - Success prompt
	WOOT = bold + green + "[$] " + normal
)

var spinner = []string{
	"⠋",
	"⠙",
	"⠹",
	"⠸",
	"⠼",
	"⠴",
	"⠦",
	"⠧",
	"⠇",
	"⠏",
}

// QuerySet - A set of base64 encoded password hashes to query with
type QuerySet struct {
	Algorithm string   `json:"algorithm"`
	Hashes    []string `json:"hashes"`
}

// Result - Single result of a given hash
type Result struct {
	Preimage string `json:"preimage"`
	Hash     string `json:"hash"`
}

// ResultSet - The set of results from a QuerySet
type ResultSet struct {
	Algorithm string   `json:"algorithm"`
	Results   []Result `json:"results"`
}

// BigRainbowError - API error
type BigRainbowError struct {
	Error string `json:"error"`
}

// BigRainbowConfig - Configuration data
type BigRainbowConfig struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

func main() {

	config, err := getConfig()
	if err != nil {
		fmt.Printf("%sError: %v", WARN, err)
		return
	}

	algorithmPtr := flag.String("a", "", "algorithm")
	encodingPtr := flag.String("e", "base64", "hash encoding")
	isFile := flag.Bool("f", false, "read hashes from file")

	flag.Usage = func() {
		fmt.Println()
		fmt.Println(bold + "Usage " + normal + "bigrainbow -a <algorithm> [options] hash...")
		flag.PrintDefaults()
	}

	flag.Parse()

	var hashes []string
	if *isFile {
		for _, hashFile := range flag.Args() {
			fmt.Printf(INFO+"Reading hashes from %s ... ", hashFile)
			data := readHashesFromFile(hashFile)
			fmt.Printf("%d line(s)\n", len(data))
			hashes = append(hashes, data...)
		}
	} else {
		hashes = flag.Args()
	}
	hashes = unique(hashes)

	if 0 < len(hashes) {
		if *encodingPtr == hexEncoding {
			hashes = hexToBase64(hashes)
		}
		bigRainbowCrack(config, (*algorithmPtr), hashes)
	}

}

// Convert hex encoded hashes to base64
func hexToBase64(hashes []string) []string {
	var base64Hashes []string
	for _, hash := range hashes {
		data, err := hex.DecodeString(hash)
		if err != nil {
			base64Hash := base64.StdEncoding.EncodeToString(data)
			base64Hashes = append(base64Hashes, base64Hash)
		}
	}
	return base64Hashes
}

// Create a config or read the existing one
func getConfig() (BigRainbowConfig, error) {
	config, err := readConfig()
	if err != nil {
		fmt.Printf("%sError: %v\n", WARN, err)
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(READ + "URL: ")
		url, _ := reader.ReadString('\n')
		fmt.Print(READ + "Key: ")
		key, _ := reader.ReadString('\n')
		config.URL = strings.TrimSpace(url)
		config.Key = strings.TrimSpace(key)
		err = writeConfig(config)
	}
	return config, err
}

// Read config from file
func readConfig() (BigRainbowConfig, error) {
	config := BigRainbowConfig{}

	usr, _ := user.Current()
	configPath := path.Join(usr.HomeDir, configDirName, configFileName)

	jsonFile, err := os.Open(configPath)
	defer jsonFile.Close()
	if err != nil {
		return config, err
	}
	bytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return config, err
	}
	json.Unmarshal(bytes, &config)
	return config, nil
}

// Write config to file and create directories
func writeConfig(config BigRainbowConfig) error {

	usr, _ := user.Current()
	configDirPath := path.Join(usr.HomeDir, configDirName)

	if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
		os.MkdirAll(configDirPath, os.ModePerm)
	}

	configPath := path.Join(configDirPath, configFileName)
	bytes, _ := json.Marshal(config)
	err := ioutil.WriteFile(configPath, bytes, 0644)
	return err
}

// Because Go doesn't have `Sets`
func unique(values []string) []string {
	uniqueValues := make(map[string]bool)
	for _, value := range values {
		if 0 < len(value) {
			uniqueValues[value] = true
		}
	}
	var keys []string
	for key := range uniqueValues {
		keys = append(keys, key)
	}
	return keys
}

// Read hashes from newline delimited file
func readHashesFromFile(filePath string) []string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf(WARN+"%v", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// Displays our terminal spinner until done channel recvs
func displaySpinner(hashCount int, done <-chan bool) {
	counter := 0
	for {
		select {
		case <-done:
			return
		case <-time.After(25 * time.Millisecond):
			fmt.Printf(clear+"%s cracking %d hashes, please wait...",
				string(spinner[counter%len(spinner)]), hashCount)
			counter++
		}
	}
}

// Display results of cracking hashes
func displayResults(querySet QuerySet, resultSet ResultSet) {
	fmt.Printf(INFO+"Cracked %d of %d hashes\n", len(resultSet.Results), len(querySet.Hashes))
	for _, result := range resultSet.Results {
		fmt.Printf(" %s -> %s \n", result.Hash, result.Preimage)
	}
}

// Crack a slice of hashings with a config
func bigRainbowCrack(config BigRainbowConfig, algorithm string, hashes []string) {

	done := make(chan bool)
	go displaySpinner(len(hashes), done)

	querySet := QuerySet{
		Algorithm: algorithm,
		Hashes:    hashes,
	}
	resultSet, err := bigRainbowQuery(config, querySet)
	done <- true
	fmt.Print(clear)
	if err != nil {
		fmt.Printf(WARN+"Error: %v", err)
	} else {
		displayResults(querySet, resultSet)
	}
}

// Send the HTTP POST request to the API Gateway/AWS Lambda
func bigRainbowQuery(config BigRainbowConfig, querySet QuerySet) (ResultSet, error) {

	reqBody, _ := json.Marshal(querySet)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", config.URL, bytes.NewBuffer(reqBody))
	req.Header.Set("x-api-key", config.Key)
	req.Header.Set("content-type", jsonMime)

	resp, _ := client.Do(req)

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		var resultSet ResultSet
		json.Unmarshal(body, &resultSet)
		return resultSet, nil
	} else if resp.StatusCode == 400 {
		var errBigRainbow BigRainbowError
		json.Unmarshal(body, &errBigRainbow)
		return ResultSet{}, errors.New(errBigRainbow.Error)
	} else {
		return ResultSet{}, fmt.Errorf("Unknown error (%d)", resp.StatusCode)
	}

}
