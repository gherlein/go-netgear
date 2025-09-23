package client

import (
	"net/http"
	"github.com/gherlein/go-netgear/internal/common"
	"github.com/gherlein/go-netgear/internal/types"
)

func RequestPage(args *types.GlobalOptions, host string, url string) (string, error) {
	return common.RequestPage(args, host, url)
}

func postPage(args *types.GlobalOptions, host string, url string, requestBody string) (string, error) {
	return common.DoHttpRequestAndReadResponse(args, http.MethodPost, host, url, requestBody)
}

func DoHttpRequestAndReadResponse(args *types.GlobalOptions, httpMethod string, host string, requestUrl string, requestBody string) (string, error) {
	return common.DoHttpRequestAndReadResponse(args, httpMethod, host, requestUrl, requestBody)
}

func DoUnauthenticatedHttpRequestAndReadResponse(args *types.GlobalOptions, httpMethod string, requestUrl string, requestBody string) (string, error) {
	return common.DoUnauthenticatedHttpRequestAndReadResponse(args, httpMethod, requestUrl, requestBody)
}
