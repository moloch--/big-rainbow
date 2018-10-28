package main

/*
	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Entry - Rainbow table entry
type Entry struct {
	Preimage string `json:"preimage"`
	Md5      string `json:"md5"`
	Sha1     string `json:"sha1"`
	Sha256   string `json:"sha256"`
	Sha512   string `json:"sha512"`
}

var fileMutex sync.Mutex

func md5Sum(preimage string) string {
	digest := md5.New()
	io.WriteString(digest, preimage)
	return fmt.Sprintf("%x", digest.Sum(nil))
}

func sha1Sum(preimage string) string {
	digest := sha1.New()
	io.WriteString(digest, preimage)
	return fmt.Sprintf("%x", digest.Sum(nil))
}

func sha256Sum(preimage string) string {
	digest := sha256.New()
	io.WriteString(digest, preimage)
	return fmt.Sprintf("%x", digest.Sum(nil))
}

func sha512Sum(preimage string) string {
	digest := sha512.New()
	io.WriteString(digest, preimage)
	return fmt.Sprintf("%x", digest.Sum(nil))
}

func computeWord(word string) ([]byte, error) {
	var entry Entry
	preimage := strings.TrimSpace(word)

	entry.Preimage = preimage
	entry.Md5 = md5Sum(preimage)
	entry.Sha1 = sha1Sum(preimage)
	entry.Sha256 = sha256Sum(preimage)
	entry.Sha512 = sha512Sum(preimage)
	return json.Marshal(entry)
}

func computeEntry(word string, output *os.File, wg *sync.WaitGroup) {
	data, err := computeWord(word)
	if err == nil {
		fileMutex.Lock()
		output.Write(data)
		output.Write([]byte("\n"))
		fileMutex.Unlock()
	}
	wg.Done()
}

func computeFile(input string, output string) {
	fInput, err := os.Open(input)
	if err != nil {
		log.Fatal(err)
	}
	defer fInput.Close()

	fOutput, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer fOutput.Close()

	var wg sync.WaitGroup
	scanner := bufio.NewScanner(fInput)
	counter := 0
	for scanner.Scan() {
		word := scanner.Text()
		wg.Add(1)

		// parallel-ish
		go computeEntry(word, fOutput, &wg)

		counter++
		fmt.Printf("\rGo compute: %d", counter)
	}
	fmt.Printf("\rWaiting for go routines to finish ... ")
	wg.Wait()
	fmt.Printf("done")
}

func main() {
	fmt.Printf(" In: %s\n", os.Args[1])
	fmt.Printf("Out: %s\n", os.Args[2])
	computeFile(os.Args[1], os.Args[2])
}
