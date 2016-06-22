package rage4

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "io/ioutil"
  "net/http"
  "net/url"
)

type Client struct {
  Email string
  AccountKey string
  Url url.URL
  Http *http.Client
}

func NewClient( email string, accountKey string) (*Client, error) {

  // create client object to use for requests
  rage4ApiUrl, _ :=  url.Parse("https://secure.rage4.com/rapi/")
  client := Client{
    AccountKey: accountKey,
    Email: email,
    Url: *rage4ApiUrl,
    Http:  http.DefaultClient,
  }

  return &client, nil
}

func (c *Client) NewRequest( body map[string]interface{}, method string, endpoint string, parameters map[string]string) (*http.Request, error) {

  u := c.Url
  u.Path = u.Path + endpoint

  v := url.Values{}
  for param,value := range parameters {
    v.Set(param, value)
  }
  u.RawQuery = v.Encode()

  fmt.Println("url = ", u.String())
    
  rBody, err := encodeBody(body)
  if err != nil {
    return nil, fmt.Errorf("Error encoding request body: %s", err)
  }

  // Build the request
  req, err := http.NewRequest(method, u.String(), rBody)
  if err != nil {
    return nil, fmt.Errorf("Error creating request: %s", err)
  }

  
  // add auth details
  req.SetBasicAuth( c.Email, c.AccountKey)

  if method != "GET" {
    req.Header.Add("Content-Type", "application/json")
  }

  return req, nil
}

// decodeBody is used to JSON decode a body
func decodeBody(resp *http.Response, out interface{}) error {
  body, err := ioutil.ReadAll(resp.Body)

  if err != nil {
    return err
  }

  if err = json.Unmarshal(body, &out); err != nil {
    return err
  }

  return nil
}

//
func encodeBody(obj interface{}) (io.Reader, error) {
  buf := bytes.NewBuffer(nil)
  enc := json.NewEncoder(buf)
  if err := enc.Encode(obj); err != nil {
    return nil, err
  }
  return buf, nil
}

//
// checkResp wraps http.Client.Do() and verifies that the
// request was successful. A non-200 request returns an error
// formatted to included any validation problems or otherwise
func checkResp(resp *http.Response, err error) (*http.Response, error) {
  // If the err is already there, there was an error higher
  // up the chain, so just return that
  if err != nil {
    return resp, err
  }

  switch i := resp.StatusCode; {
  case i == 200:
    return resp, nil
  case i == 201:
    return resp, nil
  case i == 202:
    return resp, nil
  case i == 204:
    return resp, nil
  case i == 422:
    return nil, fmt.Errorf("API Error: %s", resp.Status)
  case i == 400:
    return nil, fmt.Errorf("API Error: %s", resp.Status)
    // return nil, parseErr(resp)
  default:
    return nil, fmt.Errorf("API Error: %s", resp.Status)
  }
}

