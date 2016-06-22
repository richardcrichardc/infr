package rage4

import (
  "encoding/json"
  "io"
)

func (c *Client) GetDomains() ([]Domain, error) {

  // create http request
  req, err := c.NewRequest(nil, "GET", "getdomains", nil)
  if err != nil {
    return nil, err
  }

  // issue the API request
  resp, err := c.Http.Do(req)
  if err != nil {
    return nil, err
  }
  defer resp.Body.Close()

  // parse the response
  getDomainResponse := []Domain{}
  err = decode(resp.Body, &getDomainResponse)
  if err != nil {
    return nil, err
  }

  domains := make([]Domain, len(getDomainResponse))
  for i, domain := range getDomainResponse {
    domains[i] = domain
  }
  
  return domains, nil
}

//
func decode(reader io.Reader, obj interface{}) error {
  decoder := json.NewDecoder(reader)
  err := decoder.Decode(&obj)
  if err != nil {
    return err
  }
  return nil
}


type Domain struct {
  Id          int       `json:"id"`
  Name        string    `json:"name"`
  Email       string    `json:"owner_email"`
  Type        int       `json:"type"`
  SubnetMask  int       `json:"subnet_mask"`
  DefaultNS1  string    `json:"default_ns1"`
  DefaultNS2  string    `json:"default_ns2"`    
}

