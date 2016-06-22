package rage4

import (
  "fmt"
)

func (c *Client) GetRecords( DomainId int) ([]Record, error) {

  // create http request
  endpoint := fmt.Sprintf("getrecords/%d", DomainId)
  req, err := c.NewRequest(nil, "GET", endpoint, nil)
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
  getRecordsResponse := []Record{}
  err = decode(resp.Body, &getRecordsResponse)
  if err != nil {
    return nil, err
  }

  records := make([]Record, len(getRecordsResponse))
  for i, record := range getRecordsResponse {
    records[i] = record
  }
  
  return records, nil
}


type Record struct {
  Id                int       `json:"id"`
  Name              string    `json:"name"`
  Content           string    `json:"content"`
  Type              string    `json:"type"`
  TTL               int       `json:"ttl"`
  Priority          int       `json:"priority"`
  DomainId          int       `json:"domain_id"`
  GeoRegionId       int       `json:"geo_region_id"`
  GeoLat            float64   `json:"geo_lat"`
  GeoLong           float64   `json:"geo_long"`
  FailoverEnabled   bool      `json:"failover_enabled"`
  FailoverContent   string    `json:"failover_content"`
  FailoverWithdraw  bool      `json:"failover_withdraw"`
  IsActive          bool      `json:"is_active"`
  UdpLimit          bool      `json:"udp_limit"`
}

