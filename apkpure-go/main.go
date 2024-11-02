package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/imroc/req/v3"
)

const PAGE_COUNT = 13

func channelToFile(ch <-chan string, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating %s: %v\n", filename, err)
		return
	}
	defer file.Close()

	for message := range ch {
		_, err := file.WriteString(message + "\n")
		if err != nil {
			fmt.Printf("Error writing to %s: %v\n", filename, err)
			return
		}
	}
}

func main() {
	//c := req.DevMode()
	c := req.C()
	c.ImpersonateChrome()

	packageCh := make(chan string)
	sigCh := make(chan string)

	go channelToFile(packageCh, "output/packages.txt")
	go channelToFile(sigCh, "output/sig.txt")

	client := APKPureClient{c}

	for page := range PAGE_COUNT {
		fmt.Printf("Page: %v\n", page)
		appUrls := client.FetchAppsByDeveloper("Google LLC", page+1)
		for _, appUrl := range appUrls {
			sig := client.FetchAppSig(appUrl)
			packageName := strings.Split(appUrl, "/")[2]
			packageCh <- packageName
			if sig != "" && len(sig) == 40 {
				sigCh <- sig
			}
			fmt.Println(packageName, sig)
		}
	}

	close(packageCh)
	close(sigCh)
}
