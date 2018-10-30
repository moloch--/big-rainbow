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

	HTTPTimeout = 60

	Normal = "\033[0m"
	Black  = "\033[30m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Orange = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	Bold   = "\033[1m"
	Clear  = "\r\x1b[2K"
	UpN    = "\033[%dA"
	DownN  = "\033[%dB"

	INFO = Bold + Cyan + "[*] " + Normal
	WARN = Bold + Red + "[!] " + Normal
	READ = Bold + Purple + "[?] " + Normal
	WOOT = Bold + Green + "[$] " + Normal
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
		fmt.Println(Bold + "Usage " + Normal + "bigrainbow -a <algorithm> [options] hash...")
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

func displaySpinner(hashCount int, done <-chan bool) {
	counter := 0
	for {
		select {
		case <-done:
			return
		case <-time.After(25 * time.Millisecond):
			fmt.Printf(Clear+"%s cracking %d hashes, please wait...",
				string(spinner[counter%len(spinner)]), hashCount)
			counter++
		}
	}
}

func displayResults(querySet QuerySet, resultSet ResultSet) {
	fmt.Printf(INFO+"Cracked %d of %d hashes\n", len(resultSet.Results), len(querySet.Hashes))
	for _, result := range resultSet.Results {
		fmt.Printf(" %s -> %s \n", result.Hash, result.Preimage)
	}
}

func bigRainbowCrack(config BigRainbowConfig, algorithm string, hashes []string) {

	done := make(chan bool)
	go displaySpinner(len(hashes), done)

	querySet := QuerySet{
		Algorithm: algorithm,
		Hashes:    hashes,
	}
	resultSet, err := bigRainbowQuery(config, querySet)
	done <- true
	fmt.Print(Clear)
	if err != nil {
		fmt.Printf(WARN+"Error: %v", err)
	} else {
		displayResults(querySet, resultSet)
	}
}

func bigRainbowQuery(config BigRainbowConfig, querySet QuerySet) (ResultSet, error) {

	reqBody, _ := json.Marshal(querySet)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", config.URL, bytes.NewBuffer(reqBody))
	req.Header.Set("x-api-key", config.Key)
	req.Header.Set("Content-type", jsonMime)

	resp, _ := client.Do(req)

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		var resultSet ResultSet
		json.Unmarshal(body, &resultSet)
		return resultSet, nil
	} else if resp.StatusCode == 400 {
		var bigQueryError BigRainbowError
		json.Unmarshal(body, &bigQueryError)
		return ResultSet{}, errors.New(bigQueryError.Error)
	} else {
		return ResultSet{}, fmt.Errorf("Unknown error (%d)", resp.StatusCode)
	}

}
