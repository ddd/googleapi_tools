package main

import (
	"flag"
	"fmt"
	"gapi-service/android/auth"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	var endpoint string
	var httpMethod string
	var contentType string

	flag.StringVar(&endpoint, "e", "", "Specify the endpoint to fetch the gRPC service name and required scopes of.")
	flag.StringVar(&httpMethod, "x", "POST", "Specify the HTTP method (ex. GET/POST)")
	flag.StringVar(&contentType, "c", "json", "Specify the Content-Type (supported: json, protojson, proto)")

	flag.Parse()

	if endpoint == "" {
		panic("no endpoint specified. specify an endpoint with -e (ex. -e https://youtubei.googleapis.com/youtubei/v1/browse)")
	}

	client := &fasthttp.Client{NoDefaultUserAgentHeader: true}
	accessToken, err := auth.GetAccessToken(client, os.Getenv("ANDROID_REFRESH_TOKEN"))
	if err != nil {
		panic(err)
	}

	var ct []byte
	switch contentType {
	case "json":
		ct = []byte("application/json")
	case "protojson":
		ct = []byte("application/json+protobuf")
	case "proto":
		ct = []byte("application/x-protobuf")
	}

	scopes, respBytes, respContentType, err := FetchEndpoint(client, accessToken, ct, endpoint, strings.ToUpper(httpMethod))
	if err != nil {
		panic(err)
	}

	method, service, err := ParseErrorResponse(respBytes, respContentType)
	if err != nil {
		panic(err)
	}

	fmt.Println("scopes:", string(scopes))
	fmt.Println("method:", method)
	fmt.Println("service:", service)

}
