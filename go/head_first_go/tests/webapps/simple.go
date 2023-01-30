package main

import (
  "log"
  "net/http"
)

func viewHandler(writer http.ResponseWriter, request *http.Request) {
  message := []byte("Ciao, web!")
  _, err:=writer.Write(message)
  if err != nil {
    log.Fatal(err)
  }
}

func main() {
  http.HandleFunc("/ciao", viewHandler)
  err:=http.ListenAndServe("localhost:8080", nil)
  log.Fatal(err)
}
