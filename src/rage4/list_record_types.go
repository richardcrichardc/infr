package rage4

import (
)

func (c *Client) ListRecordTypes() ([]RecordType, error) {

  // create http request
  req, err := c.NewRequest(nil, "GET", "listrecordtypes", nil)
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
  getRecordTypeResponse := []RecordType{}
  err = decode(resp.Body, &getRecordTypeResponse)
  if err != nil {
    return nil, err
  }

  recordTypes := make([]RecordType, len(getRecordTypeResponse))
  for i, recordType := range getRecordTypeResponse {
    recordTypes[i] = recordType
  }
  
  return recordTypes, nil
}


type RecordType struct {
  Name        string    `json:"name"`
  Value       int       `json:"value"`
}

