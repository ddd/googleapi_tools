package main

import (
	"fmt"
	"regexp"

	"github.com/imroc/req/v3"
)

var (
	appsRe      = regexp.MustCompile(`<p class="search-title"><a href="([^"]*)">`)
	signatureRe = regexp.MustCompile(`Signature<\/div><div class="value double-lines">(.*)<\/div><\/div><\/li><li>`)
)

func retryHook(resp *req.Response, err error) {
	req := resp.Request.RawRequest
	fmt.Printf("DEBUG: Status: %v, Error: %v, Retrying request: %v, %v\n", resp.StatusCode, err, req.Method, req.URL)
}

func retryCondition(resp *req.Response, err error) bool {
	return err != nil || !(resp.StatusCode == 200 || resp.StatusCode == 410)
}

type APKPureClient struct {
	client *req.Client
}

// Returns list of app urls (ex. /google-translate/com.google.android.apps.translate)
func (ap APKPureClient) FetchAppsByDeveloper(developer string, page int) []string {

	resp, err := ap.client.R().SetRetryCount(5).SetRetryHook(retryHook).SetRetryCondition(retryCondition).Get(fmt.Sprintf("https://apkpure.com/developer/%s?page=%v", developer, page))
	if err != nil {
		panic(err)
	}

	appsResp := appsRe.FindAllStringSubmatch(resp.String(), -1)
	var appUrls []string
	for _, i := range appsResp {
		appUrls = append(appUrls, i[1])
	}

	return appUrls
}

// Returns SHA-1 signature of app (ex. 394d84cd2cf89d3453702c663f98ec6554afc3cd)
func (ap APKPureClient) FetchAppSig(appUrl string) string {

	resp, err := ap.client.R().SetRetryCount(5).SetRetryHook(retryHook).SetRetryCondition(retryCondition).Get(fmt.Sprintf("https://apkpure.com%s/download", appUrl))
	if err != nil {
		panic(err)
	}

	sig := signatureRe.FindStringSubmatch(resp.String())
	if sig == nil {
		return ""
	} else {
		return sig[1]
	}
}
