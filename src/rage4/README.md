This is a package to allow Go apps to use the [Rage4 DNS API](https://gbshouse.uservoice.com/knowledgebase/articles/109834-rage4-dns-developers-api)

### Usage 
Using the API is very straight forward.  Create a client object and then you can start calling the various Rage4 API commands.

Example usage:

```
	import (
		"github.com/anuaimi/rage4"
	)
	
	client := rage4.Client{
		AccountKey: "account_key",
		Email: "rage_login",
		URL:   "https://secure.rage4.com/rapi/",
		Http:  http.DefaultClient,
  }

  domains, err := client.GetDomains()
```
 
### Testing
You can run the tests by typing `go test`.  This will load the rage4 testings from `config/testing.json`.  The default file will setup a local HTTP server and run the tests using that server.  If you put actual Rage4 credentials in the `testing.json` and remove the URL field, the tests will be run against the real Rage4 API.  

### Notes 
While the code should work with Go 1.3, it is tested with Go 1.4

[![GoDoc](https://godoc.org/github.com/anuaimi/rage4?status.png)](https://godoc.org/github.com/anuaimi/rage4)
[![Build Status](https://travis-ci.org/anuaimi/rage4.svg)](https://travis-ci.org/anuaimi/rage4)
