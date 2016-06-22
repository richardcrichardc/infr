package rage4

import (
  "fmt"
)

func (c *Client) DeleteDomain(DomainId int) (status Status, err error) {

  // create http request
  endpoint := fmt.Sprintf("deletedomain/%d", DomainId)
  req, err := c.NewRequest(nil, "GET", endpoint, nil)
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




