package main

import (
	"log"
	"net/http"
	"net/url"

	"wiliam.dev/apigateway"
)

func main() {
	gateway := apigateway.New()

	target, _ := url.Parse("https://api.example.com/v1/customer-products/")
	proxy := apigateway.NewPassthroughReverseProxy(target)

	gateway.Handle("GET", "/v2/customers/products/", proxy)

	log.Fatal(http.ListenAndServe(":8080", gateway.Router()))
}
