package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)
func getLen(url string) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(body))

}
func main() {
  getLen("https://google.com")
  getLen("http://vcrini.com")
  getLen("https://ilpost.it")
}
