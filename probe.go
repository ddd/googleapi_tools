package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/valyala/fasthttp"
)

type FieldViolation struct {
	Field       string `json:"field"`
	Description string `json:"description"`
}

type ErrorResponse struct {
	Error struct {
		Details []struct {
			FieldViolations []FieldViolation `json:"fieldViolations"`
		} `json:"details"`
	} `json:"error"`
}

func testAPI(method, url string, headers map[string]string, payload []byte) (int, []byte, error) {

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	req.Header.SetMethod(method)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Content-Type", "application/json+protobuf")
	req.SetBody(payload)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := fasthttp.Do(req, resp)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode(), resp.Body(), err

}

func probeAPI(method, url string, headers map[string]string, payload []byte) ([]FieldViolation, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	req.Header.SetMethod(method)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Content-Type", "application/json+protobuf")
	req.SetBody(payload)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := fasthttp.Do(req, resp)
	if err != nil {
		return nil, err
	}

	var violations []FieldViolation

	// Content-Type: application/json+protobuf (protojson)
	if bytes.Contains(resp.Header.Peek("Content-Type"), []byte("application/json+protobuf")) {
		return nil, errors.New("protojson parsing has not been implemented yet, try ?alt=json")

	} else if bytes.Contains(resp.Header.Peek("Content-Type"), []byte("application/json")) {
		var response ErrorResponse
		err = json.Unmarshal(resp.Body(), &response)
		if err != nil {
			return nil, err
		}

		if response.Error.Details == nil {
			return nil, nil
		}
		violations = response.Error.Details[0].FieldViolations
	} else {
		return nil, fmt.Errorf("%s parsing has not been implemented yet, try ?alt=json", string(resp.Header.Peek("Content-Type")))
	}

	return violations, nil

}
