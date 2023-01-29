package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Page struct {
  URL string
  Size int
}
func getLen(url string, channel chan Page) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
  channel <- Page{Size: len(body), URL: url}

}
func main() {
	sizes := make(chan Page)
	go getLen("https://google.com", sizes)
	go getLen("http://vcrini.com", sizes)
	go getLen("https://ilpost.it", sizes)
	fmt.Println(<-sizes)
	fmt.Println(<-sizes)
	fmt.Println(<-sizes)
}
