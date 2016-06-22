package rage4

import (
  "encoding/json"
  "fmt"
  "os"
  "testing"
  "net/http"
  "net/url"
  "time"
  "github.com/spf13/viper"
)

var client Client

func TestMain(m *testing.M) {


  time.Sleep(100 * time.Millisecond)

  // get settings for rage4
  viper.SetDefault("URL", "https://secure.rage4.com/rapi/")

  viper.SetConfigName("testing")     // can be testing.json, testing.yaml
  viper.AddConfigPath("./example")
  viper.ReadInConfig()

  accountKey := viper.GetString("AccountKey")
  email := viper.GetString("Email")
  apiUrl, _ :=  url.Parse(viper.GetString("URL"))

  // if URL supplied, assume was to test with mocks
  if (accountKey == "use_mocks") {
    apiUrl, _ = url.Parse("http://localhost:9000/rapi/")

    // setup mock rage4 api server()
    go serverWebPages()
  }
  // otherwise use real API 

  fmt.Printf("testing rage4 api at %s using account %s\n", apiUrl, accountKey)

  // create client to test API calls
  client = Client{
    AccountKey: accountKey,
    Email: email,
    Url: *apiUrl,
    Http:  http.DefaultClient,
  }

  if (client == Client{}) {
    os.Exit(-1)
  }

  retCode := m.Run()

  os.Exit(retCode)
}

func serverWebPages() {

  // create web server - port 9000
  http.HandleFunc("/testAuth", testAuthHandler)

  baseURL := "/rapi/"
  http.HandleFunc( baseURL + "index", getCurrentTimeHandler)
  http.HandleFunc( baseURL + "getdomains", getDomainsHandler)
  http.HandleFunc( baseURL + "getdomain", getDomainHandler)
  http.HandleFunc( baseURL + "getdomainbyname", getDomainsHandler)
  http.HandleFunc( baseURL + "createregulardomain", createRegularDomainHandler)
  http.HandleFunc( baseURL + "createregulardomainext", createRegularDomainExtHandler)
  http.HandleFunc( baseURL + "createreversedomain4", createReverseDomain4Handler)
  http.HandleFunc( baseURL + "createreversedomain6", createReverseDomain6Handler)
  http.HandleFunc( baseURL + "updatedomain", updateDomainHandler)
  http.HandleFunc( baseURL + "deletedomain", deleteDomainHandler)
  http.HandleFunc( baseURL + "showcurrentusage", showCurrentUsageHandler)
  http.HandleFunc( baseURL + "showcurrentglobalusage", showCurrentGlobalUsageHandler)
  http.HandleFunc( baseURL + "listrecordtypes", listRecordTypesHandler)
  http.HandleFunc( baseURL + "listgeoregions", listGeoRegionsHandler)
  http.HandleFunc( baseURL + "getrecords", getRecordsHandler)
  http.HandleFunc( baseURL + "createrecord", createRecordHandler)
  http.HandleFunc( baseURL + "updaterecord", updateRecordHandler)
  http.HandleFunc( baseURL + "deleterecord", deleteRecordHandler)

  http.ListenAndServe(":9000", nil)

}

func testAuthHandler(w http.ResponseWriter, r *http.Request) {
  // test run completed
  client.GetCurrentTime()
  os.Exit(0)
}


func getCurrentTimeHandler(w http.ResponseWriter, r *http.Request) {
  // return sample time
  serverTime := ApiTime{ UtcTime: "2015-11-23T04:38:04.7781411Z", Version: "5.7.5785.25371"}
  js, err := json.Marshal(serverTime)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  w.Write( js)
}

func getDomainsHandler(w http.ResponseWriter, r *http.Request) {
  // return domain list
  domains := []Domain{ {Id: 1, Name: "domain.com", Email: "owner@rage4.com", Type: 0,
                      SubnetMask: 0, DefaultNS1: "ns1.r4ns.com", DefaultNS2: "ns2.r4ns.com"}}
  js, err := json.Marshal(domains)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  w.Write( js)
}

func getDomainHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}

func createRegularDomainHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}

func createRegularDomainExtHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}

func createReverseDomain4Handler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}

func createReverseDomain6Handler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}

func updateDomainHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}

func deleteDomainHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}

func showCurrentUsageHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}
func showCurrentGlobalUsageHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}
func listRecordTypesHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}
func listGeoRegionsHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}
func getRecordsHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}
func createRecordHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}
func updateRecordHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}
func deleteRecordHandler(w http.ResponseWriter, r *http.Request) {
  // no input required
  // return array of domain info
}

