package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
  "time"
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
  go getLen("https://google.com")
  go getLen("http://vcrini.com")
  go getLen("https://ilpost.it")
  time.Sleep(time.Second*2)
}
