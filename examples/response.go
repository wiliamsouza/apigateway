package main

import (
	"log"
	"net/http"
	"net/url"

	"wiliam.dev/apigateway"
)

func main() {
	gateway := apigateway.New()

	reponseTranslator := func(res *http.Response) error {
		// Here you implement trasnlator for HTTP response
		// let's say you need merge product information inside
		// order payload that is the place to do so.
		return nil
	}

	target, err := url.Parse("https://api.example.com/v1/orders/:code")
	if err != nil {
		log.Panic("Error parsing target url")
	}
	proxy := apigateway.NewResponseTranslatorReverseProxy(target, reponseTranslator)

	gateway.Handle("GET", "/v2/orders/:code", proxy)

	log.Fatal(http.ListenAndServe(":8080", gateway.Router()))
}
