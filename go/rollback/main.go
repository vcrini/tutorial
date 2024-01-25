package main

import (
	"net/http"

	"github.com/vcrini/go-utils"
)

func main() {
	http.HandleFunc("/versions", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, present versions are:\n"))
		buildCommand := []string{"/Users/vcrini/go/bin/aws-manager-service", "--findVersion", "dpl-app-appdemo-backend"}
		out := utils.Exe(buildCommand)
		w.Write([]byte(out))
	})
	http.ListenAndServe(":8080", nil)
}
