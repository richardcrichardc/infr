package rage4

import (
  "testing"
  "time"
)

func TestGetCurrentTime(t *testing.T) {
  //make request
  serverInfo, err := client.GetCurrentTime()
  if (err != nil) {
    t.Error("GetCurrentTime - ", err)
  }

  // check have valid time
  _, err = time.Parse(time.RFC3339Nano, serverInfo.UtcTime)
  if (err != nil) {
    t.Error("GetCurrentTime - invalid timestamp - ", err)
  }

  // make sure we got some sort of version
  if (len(serverInfo.Version) == 0) {
    t.Error("GetCurrentTime - no version returned")
  }
}
