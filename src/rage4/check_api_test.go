package rage4

import (
  "testing"
)

func TestAuthentication(t *testing.T) {

  result, _ := client.CheckAPI()

  if (result == 0) {
    t.Error("internal error")
  } else if (result == 1) {
    t.Error("server not responding")
  } else if (result == 401) {
    t.Error("invalid credentials")
  } else if (result != 200) {
    t.Error("api returned ")
  }

}
