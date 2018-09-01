package apigateway

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/julienschmidt/httprouter"
)

// Gateway API
type Gateway struct {
	router *httprouter.Router
}

// Handle registers a new request handle with the given path and method.
func (g *Gateway) Handle(method, path string, handler http.Handler) {
	g.router.Handler(method, path, handler)
}

// Router return API gateway router
func (g *Gateway) Router() *httprouter.Router {
	return g.router
}

// New API gateway
func New() *Gateway {
	return &Gateway{router: httprouter.New()}
}

// SNSReverseProxy AWS simple notification proxy
type SNSReverseProxy struct {
	TopicArn          string
	Config            *aws.Config
	RequestTranslator func(*http.Request)
}

func (p *SNSReverseProxy) errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	log.Println(err)
	rw.WriteHeader(http.StatusBadGateway)
}

func (p *SNSReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if p.RequestTranslator != nil {
		p.RequestTranslator(req)
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		defer req.Body.Close()
		panic(http.ErrAbortHandler)
	}
	req.Body.Close()

	s := session.Must(session.NewSession())

	var c *sns.SNS
	c = sns.New(s)
	if p.Config.Endpoint != (aws.Config{}.Endpoint) {
		c = sns.New(s, p.Config)
	}

	message, err := json.Marshal(map[string]string{"default": string(body)})
	if err != nil {
		p.errorHandler(rw, req, err)
	}

	input := &sns.PublishInput{
		Message:          aws.String(string(message)),
		MessageStructure: aws.String("json"),
		TopicArn:         aws.String(p.TopicArn),
	}

	ctx := req.Context()
	_, err = c.PublishWithContext(ctx, input)
	if err != nil {
		p.errorHandler(rw, req, err)
	}
	rw.WriteHeader(http.StatusAccepted)
}

// NewReverseProxy returns a new ReverseProxy that routes URLs to the scheme,
// host, and base path provided in target.
func NewReverseProxy(target *url.URL, responseTranslator func(*http.Response) error, requestTranslator func(*http.Request)) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		req.Host = target.Host
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

		targetQuery := target.RawQuery
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}

		if requestTranslator != nil {
			requestTranslator(req)
		}
	}
	if responseTranslator != nil {
		return &httputil.ReverseProxy{ModifyResponse: responseTranslator, Director: director}
	}

	return &httputil.ReverseProxy{Director: director}
}

// NewPassthroughReverseProxy redirect client call to backend target
func NewPassthroughReverseProxy(target *url.URL) *httputil.ReverseProxy {
	return NewReverseProxy(target, nil, nil)
}

// NewResponseTranslatorReverseProxy modifies response before respond it to clients
func NewResponseTranslatorReverseProxy(target *url.URL, responseTranslator func(*http.Response) error) *httputil.ReverseProxy {
	return NewReverseProxy(target, responseTranslator, nil)
}

// NewRequestTranslatorReverseProxy modifies the request into a new request before send to backend target
func NewRequestTranslatorReverseProxy(target *url.URL, requestTranslator func(*http.Request)) *httputil.ReverseProxy {
	return NewReverseProxy(target, nil, requestTranslator)
}

// NewSNSPassthroughReverseProxy publishes request body to SNS topic
func NewSNSPassthroughReverseProxy(topicArn string, config *aws.Config) *SNSReverseProxy {
	return &SNSReverseProxy{TopicArn: topicArn, Config: config}
}

// NewSNSRequestTranslatorReverseProxy publishes modified request body to SNS topic
func NewSNSRequestTranslatorReverseProxy(topicArn string, config *aws.Config, requestTranslator func(*http.Request)) *SNSReverseProxy {
	return &SNSReverseProxy{TopicArn: topicArn, Config: config, RequestTranslator: requestTranslator}
}
