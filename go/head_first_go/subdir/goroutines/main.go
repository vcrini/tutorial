package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)


func getLen(url string, channel chan int) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	channel <- len(body)

}
func main() {
	sizes := make(chan int)
	go getLen("https://google.com", sizes)
	go getLen("http://vcrini.com", sizes)
	go getLen("https://ilpost.it", sizes)
	fmt.Println(<-sizes)
	fmt.Println(<-sizes)
	fmt.Println(<-sizes)
}
