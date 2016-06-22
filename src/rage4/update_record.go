package rage4

import (
  "fmt"
  "strconv"
)

// NOTE: NOT YET WORKING!

func (c *Client) UpdateRecord( recordId int, record Record) (status Status, err error) {

  // create http request
  endpoint := fmt.Sprintf("updaterecord/%d", recordId)
  idSetting := strconv.Itoa(record.Id)
  activeSetting := strconv.FormatBool(record.IsActive)
  prioritySetting := strconv.Itoa(record.Priority)
  ttlSetting := strconv.Itoa(record.TTL)
  geozoneSetting := strconv.Itoa(record.GeoRegionId)
  
  parameters := map[string]string {
    "id" : idSetting,
    "name" : record.Name,
    "content" : record.Content,
    "priority" : prioritySetting,
    "active" : activeSetting,
    "ttl" : ttlSetting,
    "geozone" : geozoneSetting,
  }

  fmt.Printf("%s\n",endpoint)
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





