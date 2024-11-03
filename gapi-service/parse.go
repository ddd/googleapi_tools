package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"

	rpc "gapi-service/rpc"

	"google.golang.org/protobuf/proto"
)

type ErrorInfo struct {
	Reason   string `json:"reason"`
	Domain   string `json:"domain"`
	Metadata struct {
		Service string `json:"service"`
		Method  string `json:"method"`
	}
}

// JSON format of error response
type Response struct {
	Error struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Status  string      `json:"status"`
		Details []ErrorInfo `json:"details"`
	} `json:"error"`
}

var (
	methodRe  = regexp.MustCompile(`\["method",\s*"([^"]*)"\]`)
	serviceRe = regexp.MustCompile(`\["service",\s*"([^"]*)"\]`)
)

func ParseErrorResponse(respBytes []byte, contentType []byte) (method string, service string, err error) {

	// Parse protojson (JSPB) response
	if bytes.Contains(contentType, []byte("application/json+protobuf")) {

		methodReMatch := methodRe.FindSubmatch(respBytes)
		serviceReMatch := serviceRe.FindSubmatch(respBytes)
		return string(methodReMatch[1]), string(serviceReMatch[1]), nil

	} else if bytes.Contains(contentType, []byte("application/json")) {
		// Parse JSON response

		var response Response

		err := json.Unmarshal(respBytes, &response)
		if err != nil {
			return "", "", err
		}

		return response.Error.Details[0].Metadata.Method, response.Error.Details[0].Metadata.Service, nil

	} else if bytes.Contains(contentType, []byte("application/x-protobuf")) {
		// Parse PROTO response

		var response rpc.Error

		err := proto.Unmarshal(respBytes, &response)
		if err != nil {
			return "", "", err
		}

		var service string
		var method string
		for k, v := range response.ErrorDetails.ErrorInfo.Metadata {
			if k == "service" {
				service = v
			} else if k == "method" {
				method = v
			}
		}

		return method, service, nil
	}

	return "", "", errors.New("unknown content type")

}
