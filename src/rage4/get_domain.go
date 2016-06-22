package rage4

import (
  "fmt"
)

func (c *Client) GetDomain(DomainId int) (domain Domain, err error) {

  // create http request
  endpoint := fmt.Sprintf("getdomain/%d", DomainId)
  req, err := c.NewRequest(nil, "GET", endpoint, nil)
  if err != nil {
    return Domain{}, err
  }

  // issue the API request
  resp, err := c.Http.Do(req)
  if err != nil {
    return Domain{}, err
  }
  defer resp.Body.Close()

  // parse the response
  getDomainResponse := Domain{}
  err = decode(resp.Body, &getDomainResponse)
  if err != nil {
    return Domain{}, err
  }

  domain = getDomainResponse
  
  return domain, nil
}




