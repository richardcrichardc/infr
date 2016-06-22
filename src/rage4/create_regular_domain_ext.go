package rage4

import (
  // "fmt"
  // "net/url"
)

func (c *Client) CreateRegularDomainExt(Name string, Email string) (status Status, err error) {

  // Name        string    `json:"name"`
  // Email       string    `json:"owner_email"`
  // Type        int       `json:"type"`
  // SubnetMask  int       `json:"subnet_mask"`
  // DefaultNS1  string    `json:"default_ns1"`
  // DefaultNS2  string    `json:"default_ns2"`    


  // create http request
  parameters := map[string]string {
    "name" : Name,
    "email" : Email,
  }
  req, err := c.NewRequest(nil, "GET", "createregulardomainext", parameters)
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





