package rage4

import (
  // "fmt"
  "strconv"
)

func (c *Client) CreateRecord(domainId int, record Record) (status Status, err error) {

  // create http request
  parameters := map[string]string{
    "id" : strconv.Itoa(domainId),
    "name" : record.Name,
    "content" : record.Content,
    "type" : record.Type,
    "priority" : strconv.Itoa(record.Priority),
  }

  req, err := c.NewRequest(nil, "GET", "createrecord", parameters)
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






