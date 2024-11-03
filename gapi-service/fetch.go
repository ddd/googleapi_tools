package main

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/valyala/fasthttp"
)

var (
	scopesRe = regexp.MustCompile(`scope="([^"]*)"`)
)

func FetchEndpoint(client *fasthttp.Client, accessToken []byte, contentType []byte, endpoint string, method string) (scopes []byte, respBytes []byte, respContentType []byte, err error) {

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	// User-Agent isn't needed, we just add it in case of edge cases
	req.Header.SetBytesKV([]byte("User-Agent"), []byte("Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:127.0) Gecko/20100101 Firefox/127.0"))
	req.Header.SetBytesKV([]byte("Content-Type"), contentType)
	req.Header.SetBytesKV([]byte("Authorization"), accessToken)

	req.Header.SetMethod(method)
	req.Header.SetRequestURI(endpoint)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = client.Do(req, resp)
	if err != nil {
		return nil, nil, nil, err
	}

	switch resp.StatusCode() {

	// this should be the right status code when authorization fails due to insufficient scopes
	case fasthttp.StatusForbidden:
		match := scopesRe.FindSubmatch(resp.Header.Peek("Www-Authenticate"))
		if len(match) == 0 {
			return nil, resp.Body(), resp.Header.Peek("Content-Type"), fmt.Errorf("unable to parse scopes from Www-Authenticate header: %v", string(resp.Header.Peek("Www-Authenticate")))
		}

		return match[1], resp.Body(), resp.Header.Peek("Content-Type"), nil

	// this happens when the wrong contentType is given and that contentType is not configured for the server. It could also be something else, but we assume this and attempt other content-types.
	case fasthttp.StatusBadRequest:
		fmt.Println(string(resp.Body()))
		return nil, nil, nil, errors.New("content-type not configured for server")

	case fasthttp.StatusNotFound:
		return nil, nil, nil, errors.New("404 endpoint not found")

	default:
		return nil, nil, nil, fmt.Errorf("unknown status code %v", resp.StatusCode())
	}

}
