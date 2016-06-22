package rage4

import (
  "errors"
)

// make a simple API call
// see if server listening & credentials valid
func (c *Client) CheckAPI() (int, error) {

  // create http request
  req, err := c.NewRequest(nil, "GET", "index", nil)
  if err != nil {
    return 0, err
  }

  // issue the API request
  resp, err := c.Http.Do(req)
  if err != nil {
    return 1, err
  }
  defer resp.Body.Close()

  status := resp.StatusCode
  switch {
    case (status >= 300) && (status <= 399):
      return status, errors.New("redirection")
    case (status >= 400) && (status <= 499):
      return status, errors.New("client error")
    case (status >= 500) && (status <= 599):
      return status, errors.New("server error")
  }
  return status, nil
}

