package rage4

import (
)

func (c *Client) ShowCurrentGlobalUsage() (usage GlobalUsage, err error) {

  // create http request
  req, err := c.NewRequest(nil, "GET", "showcurrentglobalusage", nil)
  if err != nil {
    return GlobalUsage{}, err
  }

  // issue the API request
  resp, err := c.Http.Do(req)
  if err != nil {
    return GlobalUsage{}, err
  }
  defer resp.Body.Close()

  // parse the response
  getUsageResponse := []GlobalUsage{}
  err = decode(resp.Body, &getUsageResponse)
  if err != nil {
    return GlobalUsage{}, err
  }

  usage = getUsageResponse[0]
  
  return usage, nil
}


type GlobalUsage struct {
  Date    string    `json:"date"`
  Value   int       `json:"value"`
}

