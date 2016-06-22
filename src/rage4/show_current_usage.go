package rage4

import (
  "fmt"
)

func (c *Client) ShowCurrentUsage( DomainId int) (usage []DailyUsage, err error) {

  // create http request
  endpoint := fmt.Sprintf("showcurrentusage/%d", DomainId)
  req, err := c.NewRequest(nil, "GET", endpoint, nil)
  if err != nil {
    return []DailyUsage{}, err
  }

  // issue the API request
  resp, err := c.Http.Do(req)
  if err != nil {
    return []DailyUsage{}, err
  }
  defer resp.Body.Close()

  // parse the response
  getUsageResponse := []DailyUsage{}
  err = decode(resp.Body, &getUsageResponse)
  if err != nil {
    return []DailyUsage{}, err
  }

  usage = getUsageResponse
  
  return usage, nil
}


type DailyUsage struct {
  Date    string    `json:"date_created"`
  Total   int       `json:"total"`
  EUTotal   int       `json:"eu"`
  USTotal   int       `json:"us"`
  SATotal   int       `json:"sa"`
  APTotal   int       `json:"ap"`
  AFTotal   int       `json:"af"`
}

