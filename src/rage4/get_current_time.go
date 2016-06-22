package rage4

import (
  "errors"
  "fmt"
  // "io/ioutil"
)

func (c *Client) GetCurrentTime() (ApiTime, error) {

  // create http request
  req, err := c.NewRequest(nil, "GET", "index", nil)
  if err != nil {
    return ApiTime{}, err
  }

  // issue the API request
  resp, err := c.Http.Do(req)
  if err != nil {
    return ApiTime{}, err
  }
  defer resp.Body.Close()

  if (resp.StatusCode != 200) {
    msg := fmt.Sprintf( "request returned error %d code ", resp.StatusCode)
    return ApiTime{}, errors.New(msg)
  }

  // sample response
  // {"utctime":"2015-11-21T03:18:59.4251571Z","version":"5.7.5785.25371"}

  // parse the response
  currentServerTime := ApiTime{}
  err = decode(resp.Body, &currentServerTime)
  if err != nil {
    return ApiTime{}, err
  }
  
  return currentServerTime, nil
}


// server time (UTC)
type ApiTime struct {
  UtcTime     string    `json:"utctime"`
  Version     string    `json:"version"`
}

