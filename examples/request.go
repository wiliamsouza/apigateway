package main

import (
	"log"
	"net/http"
	"net/url"

	"wiliam.dev/apigateway"
)

func main() {
	gateway := apigateway.New()

	requestTranslator := func(req *http.Request) {
		// Here you implement translator for the given request
		// use your imagination many things can be done here
		// like payload fields rename, removal and adition.
	}

	target, err := url.Parse("https://api.example.com/v1/products/")
	if err != nil {
		log.Panic("Error parsing target url")
	}
	proxy := apigateway.NewRequestTranslatorReverseProxy(target, requestTranslator)

	gateway.Handle("GET", "/v2/products/", proxy)

	log.Fatal(http.ListenAndServe(":8080", gateway.Router()))
}
