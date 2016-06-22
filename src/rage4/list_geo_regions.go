package rage4

import (
)

func (c *Client) ListGeoRegions() ([]Region, error) {

  // create http request
  req, err := c.NewRequest(nil, "GET", "listgeoregions", nil)
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
  getRegionsResponse := []Region{}
  err = decode(resp.Body, &getRegionsResponse)
  if err != nil {
    return nil, err
  }

  regions := getRegionsResponse
  // regions := make([]Regions, len(getRegionsResponse))
  // for i, recordType := range getRecordTypeResponse {
  //   recordTypes[i] = recordType
  // }
  
  return regions, nil
}


type Region struct {
  Name        string    `json:"name"`
  Value       int       `json:"value"`
}

