apigateway
==========

apigateway is a programmable API gateway micro framework

Features
--------

* Passthrough
* Request translator
* Response translator
* AWS SNS passthrough
* AWS SNS request translator

Installation
------------

```bash
go get github.com/wiliamsouza/apigateway
```

Tests
-----

```bash
make test
```

Usage
-----

```go
package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/wiliamsouza/apigateway"
)

func main() {
	gateway := apigateway.New()

	target, _ := url.Parse("https://ifconfig.co/")
	proxy := apigateway.NewPassthroughReverseProxy(target)

	gateway.Handle("GET", "/myip", proxy)

	log.Fatal(http.ListenAndServe(":8080", gateway.Router()))
}
```

The above code is a simple passthrough gateway.

```bash
curl http://127.0.0.1:8080/myip
```

Check `examples` folder for more.
