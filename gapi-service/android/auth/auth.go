package auth

import (
	"fmt"
	"regexp"

	"github.com/valyala/fasthttp"
)

var (
	authTokenRe = regexp.MustCompile("Auth=(.*)")
)

// Fetches access token with xapi.zoo scopes given an android refesh token (aas_et/AK...)
func GetAccessToken(client *fasthttp.Client, refreshToken string) ([]byte, error) {

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.Header.SetBytesKV([]byte("User-Agent"), []byte("GoogleAuth/1.4"))
	req.Header.SetBytesKV([]byte("Content-Type"), []byte("application/x-www-form-urlencoded"))
	req.Header.SetMethod("POST")

	// use android.googleapis.com instead of android.clients.google.com for ipv6-only support
	req.Header.SetRequestURI("https://android.googleapis.com/auth")
	req.SetBodyString(fmt.Sprintf("service=oauth2:https://www.googleapis.com/auth/xapi.zoo&Token=%v", refreshToken))

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.Do(req, resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("unknown status code %v", resp.StatusCode())
	}

	match := authTokenRe.FindSubmatch(resp.Body())

	return append([]byte("Bearer "), match[1]...), nil

}
