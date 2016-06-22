package main 

import (
  "fmt"
  "net/http"
  "net/url"
  "os"
  "time"
  "github.com/anuaimi/rage4"
  "github.com/spf13/viper"
)

//
// main - start of app
//
func main() {

  // get API values from config file
  viper.SetConfigName("athir") 
  // viper.SetConfigName("config")     // can be config.json, config.yaml

  viper.AddConfigPath("./")
  viper.SetDefault("URL", "https://secure.rage4.com/rapi/")
  viper.ReadInConfig()

  accountKey := viper.GetString("AccountKey")
  email := viper.GetString("Email")

  //make sure we were able to read values
  if (len(accountKey) == 0) || (len(email) == 0) {
    fmt.Print("could not read rag4 config values\n")
    os.Exit(-1)
  }

  // create connection to Rage4
  apiUrl, _ := url.Parse(viper.GetString("URL"))
  client := rage4.Client{
    AccountKey: viper.GetString("AccountKey"),
    Email: viper.GetString("Email"),
    Url:  *apiUrl,
    Http:  http.DefaultClient,
  }

  // GetCurrentTime
  fmt.Print("GetCurrentTime\n")
  apiTime, _ := client.GetCurrentTime()
  serverTime, _ := time.Parse(time.RFC3339Nano, apiTime.UtcTime)

  fmt.Printf("  time: %s  version: %s\n", serverTime, apiTime.Version)


  // // GetDomains
  // fmt.Print("GetDomains\n")
  // domains, _ := client.GetDomains()
  // for _, domain := range domains {
  //   fmt.Printf("  %s (%d)\n", domain.Name, domain.Id)
  // }

  // //ShowCurrentGlobalUsage
  // fmt.Print("\nShowCurrentGlobalUsage\n")
  // usage,  _ := client.ShowCurrentGlobalUsage()
  // fmt.Printf("  usage from %s: %d\n", usage.Date, usage.Value)

  // //ListRecordTypes
  // fmt.Print("\nListRecordTypes\n")
  // recordTypes, err := client.ListRecordTypes()
  // if (recordTypes == nil) {
  //   fmt.Printf("error: %s", err)
  // } else {
  //   for _, recordType := range recordTypes {
  //     fmt.Printf("  %s: %d\n", recordType.Name, recordType.Value)
  //   }
  // }

  // //ListGeoRegions
  // fmt.Print("\nListGeoRegions\n")
  // regions, err := client.ListGeoRegions()
  // if (regions == nil) {
  //   fmt.Printf("error: %s", err)
  // } else {
  //   for _, region := range regions {
  //     fmt.Printf("  %s: %d\n", region.Name, region.Value)
  //   }
  // }


  //CreateRegularDomain
  var domainName = "blabla.com"

  var status rage4.Status
  var domainId int

  // see if domain exists before try and create
  fmt.Print("\nGetDomainByName\n")
  domain, err := client.GetDomainByName( domainName)
  if (domain.Id == 0) {
    // domain does not yet exists, so create
    fmt.Print("\nCreateRegularDomain\n")
    status, err = client.CreateRegularDomain( domainName, "admin@blabla.com")
    if (status.Id == 0) {
      fmt.Printf("create regular domain failed: %s\n", status.Error)
      os.Exit(-1)
    } 
    fmt.Printf("  status: %t  id: %d err:%s\n", status.Status, status.Id, status.Error)
      
    domainId = status.Id
  } else {
    fmt.Printf("  found domain %d\n", domain.Id)
    domainId = domain.Id
  }

  //GetDomain
  // fmt.Print("\nGetDomain\n")
  // domain, _ = client.GetDomain( domainId)
  // fmt.Printf("  %d: %s %s\n", domainId, domain.Name, domain.Email)


  // //UpdateDomain
  fmt.Print("\nUpdateDomain\n")
  status, err = client.UpdateDomain2( domainId, "admin@test.com", true, "ns1.blabla.com", "ns2.blabla.com")
  if (err == nil) {
    fmt.Printf("  domain updated\n")
  } else {
    fmt.Printf("%s", err)
  }

  // //ShowCurrentUsage
  // fmt.Print("\nShowCurrentUsage\n")
  // fmt.Printf("  for: %d\n", domainId)
  // domainUsage,  _ := client.ShowCurrentUsage( domainId)
  // for _, dailyUsage := range domainUsage {
  //   fmt.Printf("  from %s:  -  ", dailyUsage.Date)
  //   fmt.Printf("  Total: %d", dailyUsage.Total)
  //   fmt.Printf("  EU: %d", dailyUsage.EUTotal)
  //   fmt.Printf("  US: %d", dailyUsage.USTotal)
  //   fmt.Printf("  SA: %d", dailyUsage.SATotal)
  //   fmt.Printf("  AP: %d", dailyUsage.APTotal)
  //   fmt.Printf("  AF: %d\n", dailyUsage.AFTotal)
  // }

  // //GetRecords
  // fmt.Print("\nGetRecords\n")
  // records, err := client.GetRecords( domainId)
  // if (err != nil) {
  //   fmt.Printf("%s", err)
  // } else {
  //   for _, record := range records {
  //     fmt.Printf("  %d: %s %s\n", record.Id, record.Type, record.Content)
  //   }
  // }

  // CreateRecord
  var recordId int
  fmt.Print("\nCreateRecord\n")
  record := rage4.Record{ Name: "www.blabla.com", Content: "1.2.3.4", Type: "A", Priority: 1 }
  status, err = client.CreateRecord( domainId, record)
  if (err == nil) {
    recordId = status.Id
    fmt.Printf("  status: %t  id: %d  error: %s\n", status.Status, status.Id, status.Error)
  } else {
    fmt.Printf("  status: %t  id: %d  error: %s\n", status.Status, status.Id, status.Error)
  }

  //DeleteRecord
  if (recordId > 0) {
    fmt.Print("\nDeleteRecord\n")
    status,  err = client.DeleteRecord( recordId)
    if (err == nil) {
      fmt.Println("  record deleted")
    } else {
      fmt.Printf("  status: %t  id: %d  error: %s\n", status.Status, status.Id, status.Error)
    }    
  }

  //DeleteDomain
  fmt.Print("\nDeleteDomain\n")
  status, err = client.DeleteDomain( domainId)
  if (err == nil) {
    fmt.Println("  domain deleted")
  } else {
    fmt.Printf("  status: %t  id: %d  error: %s\n", status.Status, status.Id, status.Error)
  }

  // done

}
