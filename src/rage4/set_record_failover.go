package rage4

import (
  "fmt"
)

func (c *Client) SetRecordFailover( recordId int, failover bool) (status Status, err error) {

  // create http request
  endpoint := fmt.Sprintf("setrecordfailover/%d?", recordId)
  setting := "true"
  if failover == false {
    setting = "false"
  }
  parameters := map[string]string {
    "active" : setting,
  }
 
  req, err := c.NewRequest(nil, "GET", endpoint, parameters)
  if err != nil {
    return Status{}, err
  }

  // issue the API request
  resp, err := c.Http.Do(req)
  if err != nil {
    return Status{}, err
  }
  defer resp.Body.Close()

  // parse the response
  getStatusResponse := Status{}
  err = decode(resp.Body, &getStatusResponse)
  if err != nil {
    return Status{}, err
  }

  status = getStatusResponse
  
  return status, nil
}


