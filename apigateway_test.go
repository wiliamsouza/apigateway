package apigateway

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/stretchr/testify/assert"
)

func TestReverseProxy(t *testing.T) {

	t.Run("Test passthrough reverse proxy", func(t *testing.T) {
		// Backend API
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "backend")
		}))
		defer backend.Close()

		target, err := url.Parse(backend.URL)
		assert.Nil(t, err)

		proxy := NewPassthroughReverseProxy(target)

		// Test gateway
		gateway := httptest.NewServer(proxy)
		defer gateway.Close()

		// Request made for the API gateway should be proxied to backend API endpoint
		res, err := http.Get(gateway.URL)
		assert.Nil(t, err)

		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err)
		res.Body.Close()

		assert.Equal(t, "backend\n", string(body))

	})

	t.Run("Test response translator reverse proxy", func(t *testing.T) {
		orderPayload := "{\"id\":\"3d58fd0c-0f1e-4d75-8961-873508095817\",\"customer\":{\"id\":\"b0f956d2-bc16-4275-8980-3442e97785f7\"},\"product\":{\"sku\": \"IPOD2008GREEN\"},\"quantity\":1}"
		// Order API
		order := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, orderPayload)
		}))
		defer order.Close()

		mergedPayload := "{\"id\": \"3d58fd0c-0f1e-4d75-8961-873508095817\",\"customer\": {\"id\": \"b0f956d2-bc16-4275-8980-3442e97785f7\",\"name\": \"John Doe\",\"address\": \"123 Main St\",\"city\": \"Anytown\",\"country\": \"Brazil\"},\"product\": {\"sku\": \"IPOD2008GREEN\",\"title\": \"IPod Nano - 8gb\",\"name\": \"IPod Nano - 8gb - green\"},\"quantity\": 1}"

		reponseTranslator := func(r *http.Response) error {
			body, err := ioutil.ReadAll(r.Body)
			assert.Nil(t, err)
			r.Body.Close()
			assert.Equal(t, orderPayload+"\n", string(body))

			r.Body = ioutil.NopCloser(bytes.NewReader([]byte(mergedPayload)))
			r.ContentLength = int64(len(mergedPayload))
			r.Header.Set("Content-Length", strconv.Itoa(len(mergedPayload)))
			return nil
		}

		orderURL, err := url.Parse(order.URL)
		assert.Nil(t, err)

		proxy := NewResponseTranslatorReverseProxy(orderURL, reponseTranslator)

		// Test gateway
		gateway := httptest.NewServer(proxy)
		defer gateway.Close()

		// Request made for the API gateway should be proxied to backend API endpoint
		res, err := http.Get(gateway.URL)
		assert.Nil(t, err)

		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err)
		res.Body.Close()

		assert.Equal(t, mergedPayload, string(body))

	})

	t.Run("Test request translator reverse proxy", func(t *testing.T) {
		// Product API
		product := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			assert.Nil(t, err)
			r.Body.Close()
			fmt.Fprintln(w, string(body))
		}))
		defer product.Close()

		replacedPayload := "changed"

		requestTranslator := func(r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			assert.Nil(t, err)
			r.Body.Close()
			assert.Equal(t, "original", string(body))

			r.Body = ioutil.NopCloser(bytes.NewReader([]byte(replacedPayload)))
			r.ContentLength = int64(len(replacedPayload))
			r.Header.Set("Content-Length", strconv.Itoa(len(replacedPayload)))
		}

		productURL, err := url.Parse(product.URL)
		assert.Nil(t, err)

		proxy := NewRequestTranslatorReverseProxy(productURL, requestTranslator)

		// Test gateway
		gateway := httptest.NewServer(proxy)
		defer gateway.Close()

		// Request made for the API gateway should be changed and proxied to backend API endpoint
		res, err := http.Post(gateway.URL, "application/json", bytes.NewReader([]byte("original")))
		assert.Nil(t, err)

		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err)
		res.Body.Close()

		assert.Equal(t, replacedPayload+"\n", string(body))

	})
	t.Run("Test SNS passthrough reverse proxy", func(t *testing.T) {
		var snsRequest string
		endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := ioutil.ReadAll(r.Body)
			r.Body.Close()
			snsRequest = string(body)
			w.WriteHeader(http.StatusOK)
		}))
		defer endpoint.Close()

		topicArn := "arn:aws:sns::000000000000:api_gateway_topic"
		c := aws.NewConfig()
		c.WithRegion("us-east-1")
		c.WithCredentials(credentials.NewStaticCredentials("id", "key", ""))
		c.WithDisableSSL(true)
		c.WithEndpoint(endpoint.URL)

		proxy := NewSNSPassthroughReverseProxy(topicArn, c)

		// Test gateway
		gateway := httptest.NewServer(proxy)
		defer gateway.Close()

		// Request made for the API gateway should be proxied to SNS
		res, err := http.Post(gateway.URL, "application/json", bytes.NewReader([]byte("test_sns_sqs")))
		assert.Nil(t, err)
		assert.Equal(t, http.StatusAccepted, res.StatusCode)

		assert.Equal(t, snsRequest, "Action=Publish&Message=%7B%22default%22%3A%22test_sns_sqs%22%7D&MessageStructure=json&TopicArn=arn%3Aaws%3Asns%3A%3A000000000000%3Aapi_gateway_topic&Version=2010-03-31")
	})

	t.Run("Test SNS request translator reverse proxy", func(t *testing.T) {
		var snsRequest string
		endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := ioutil.ReadAll(r.Body)
			r.Body.Close()
			snsRequest = string(body)
			w.WriteHeader(http.StatusOK)
		}))
		defer endpoint.Close()

		requestTranslator := func(r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			assert.Nil(t, err)
			r.Body.Close()
			assert.Equal(t, "test_sns_sqs", string(body))

			changedPayload := "changed_test_sns_sqs"
			r.Body = ioutil.NopCloser(bytes.NewReader([]byte(changedPayload)))
			r.ContentLength = int64(len(changedPayload))
			r.Header.Set("Content-Length", strconv.Itoa(len(changedPayload)))
		}

		topicArn := "arn:aws:sns::000000000000:api_gateway_topic"
		c := aws.NewConfig()
		c.WithRegion("us-east-1")
		c.WithCredentials(credentials.NewStaticCredentials("id", "key", ""))
		c.WithDisableSSL(true)
		c.WithEndpoint(endpoint.URL)

		proxy := NewSNSRequestTranslatorReverseProxy(topicArn, c, requestTranslator)

		// Test gateway
		gateway := httptest.NewServer(proxy)
		defer gateway.Close()

		// Request made for the API gateway should be proxied to SNS
		res, err := http.Post(gateway.URL, "application/json", bytes.NewReader([]byte("test_sns_sqs")))
		assert.Nil(t, err)
		assert.Equal(t, http.StatusAccepted, res.StatusCode)

		assert.Equal(t, snsRequest, "Action=Publish&Message=%7B%22default%22%3A%22changed_test_sns_sqs%22%7D&MessageStructure=json&TopicArn=arn%3Aaws%3Asns%3A%3A000000000000%3Aapi_gateway_topic&Version=2010-03-31")
	})
}
