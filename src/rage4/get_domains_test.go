package rage4 

import (
  "testing"
)

func TestGetDomains(t *testing.T) {

  _, err:= client.GetDomains()
  if (err != nil) {
    t.Error("Error calling GetDomains() - ", err)
  }

}


