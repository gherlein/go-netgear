package common

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"github.com/gherlein/go-netgear/internal/types"
)

func RequestPage(args *types.GlobalOptions, host string, url string) (string, error) {
	return DoHttpRequestAndReadResponse(args, http.MethodGet, host, url, "")
}

func DoHttpRequestAndReadResponse(args *types.GlobalOptions, httpMethod string, host string, requestUrl string, requestBody string) (string, error) {
	model, token, err := ReadTokenAndModel2GlobalOptions(args, host)
	if err != nil {
		return "", err
	}

	if args.Verbose {
		fmt.Println(fmt.Sprintf("send HTTP %s request to: %s", httpMethod, requestUrl))
	}

	if IsModel316(model) {
		if strings.Contains(requestUrl, "?") {
			splits := strings.Split(requestUrl, "?")
			requestUrl = splits[0] + "?Gambit=" + token + "&" + splits[1]
		} else {
			requestUrl = requestUrl + "?Gambit=" + token
		}
	}

	req, err := http.NewRequest(httpMethod, requestUrl, strings.NewReader(requestBody))
	if err != nil {
		return "", err
	}

	if IsModel30x(model) {
		req.Header.Set("Cookie", "SID="+token)
	} else if IsModel316(model) {
		req.Header.Set("Cookie", "gambitCookie="+token)
	} else {
		panic("model not supported")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if args.Verbose {
		fmt.Println(resp.Status)
	}
	bytes, err := io.ReadAll(resp.Body)
	return string(bytes), err
}

func DoUnauthenticatedHttpRequestAndReadResponse(args *types.GlobalOptions, httpMethod string, requestUrl string, requestBody string) (string, error) {
	if args.Verbose {
		fmt.Println("Fetching data from: " + requestUrl)
	}

	req, err := http.NewRequest(httpMethod, requestUrl, strings.NewReader(requestBody))
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if args.Verbose {
		fmt.Println(resp.Status)
		for name, values := range resp.Header {
			for _, value := range values {
				fmt.Println(fmt.Sprintf("Response header: '%s' -- '%s'", name, value))
			}
		}
	}
	bytes, err := io.ReadAll(resp.Body)
	return string(bytes), err
}

func CheckIsLoginRequired(httpResponseBody string) bool {
	return len(httpResponseBody) < 10 ||
		strings.Contains(httpResponseBody, "/login.cgi") ||
		strings.Contains(httpResponseBody, "/wmi/login") ||
		strings.Contains(httpResponseBody, "/redirect.html")
}