package rage4

import (
  "fmt"
  // "net/url"
)

func (c *Client) CreateRegularDomain(Name string, Email string) (status Status, err error) {

  // create http request
  parameters := map[string]string{
    "name" : Name,
    "email" : Email,
  }

//add ability to pass parameters as map
  req, err := c.NewRequest(nil, "GET", "createregulardomain", parameters)
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
    fmt.Println("decode failed:", err)
    return Status{}, err
  }

  status = getStatusResponse
  
  return status, nil
}

type Status struct {
  Status      bool      `json:"status"`
  Id          int       `json:"id"`
  Error       string    `json:"error"`
}





