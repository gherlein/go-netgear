package models

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"github.com/gherlein/go-netgear/internal/types"
)

func DetectNetgearModel(args *types.GlobalOptions, host string) (types.NetgearModel, error) {
	url := fmt.Sprintf("http://%s/", host)
	if args.Verbose {
		fmt.Println("detecting Netgear switch model: " + url)
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	if args.Verbose {
		fmt.Println(fmt.Sprintf("HTTP response code %d", resp.StatusCode))
	}
	if resp.StatusCode != 200 {
		fmt.Println(fmt.Sprintf("Warning: response code was not 200; unusual, but will attempt detection anyway"))
	}
	responseBody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return "", err
	}
	model := detectNetgearModelFromResponse(string(responseBody))
	if model == "" {
		return "", errors.New("Can't auto-detect Netgear model from response. You may try using --model parameter ")
	}
	if args.Verbose {
		fmt.Println(fmt.Sprintf("Detected model %s", model))
	}
	return model, nil
}

func detectNetgearModelFromResponse(body string) types.NetgearModel {
	if strings.Contains(strings.ToLower(body), "<title>") && strings.Contains(body, "GS316EPP") {
		return types.GS316EPP
	}
	if strings.Contains(strings.ToLower(body), "<title>") && strings.Contains(body, "GS316EP") {
		return types.GS316EP
	}
	if strings.Contains(strings.ToLower(body), "<title>") && strings.Contains(body, "Redirect to Login") {
		return types.GS30xEPx
	}
	return ""
}
