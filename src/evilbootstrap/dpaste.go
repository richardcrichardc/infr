package evilbootstrap

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"fmt"
	"strings"
)

func Paste(fileContent string) (string, error) {
	resp, err := http.PostForm("http://dpaste.com/api/v2/",
		url.Values{
			"content": {fileContent},
			"expire_days": {"1"}})

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return "", fmt.Errorf("Unexpected HTTP response from dpaste.com: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body))+".txt", nil
}