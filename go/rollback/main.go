package main

import (
	"github.com/vcrini/go-utils"
	"net/http"
)

func main() {
	http.HandleFunc("/versions", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, present versions are"))
		buildCommand := []string{"/Users/vcrini/go/bin/aws-manager-service", "--findVersion", "dpl-app-appdemo-backend"}
		out := utils.Exe(buildCommand)
		w.Write([]byte(out))
	})
	http.ListenAndServe(":8080", nil)
}
