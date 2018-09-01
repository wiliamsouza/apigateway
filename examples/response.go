package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/wiliamsouza/apigateway"
)

func main() {
	gateway := apigateway.New()

	reponseTranslator := func(res *http.Response) error {
		// Here you implement trasnlator for HTTP response
		// let's say you need merge product information inside
		// order payload that is the place to do so.
		return nil
	}

	target, _ := url.Parse("https://api.example.com/v1/orders/:code")
	proxy := apigateway.NewResponseTranslatorReverseProxy(target, reponseTranslator)

	gateway.Handle("GET", "/v2/orders/:code", proxy)

	log.Fatal(http.ListenAndServe(":8080", gateway.Router()))
}
