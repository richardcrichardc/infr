package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type DNSVultr struct {
	apiKey string
	client http.Client
}

type vultrRecord struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Data     string `json:"data"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
	RECORDID int
}

func newDNSVultr() dnsProvider {
	var p DNSVultr

	p.apiKey = needGeneralConfig("dnsVultrAPIKey")
	p.client.Timeout = 5 * time.Second

	return &p
}

func stripZone(fqdn, zone string) string {
	if !strings.HasSuffix(fqdn, zone) {
		panic(fqdn + " is not in the zone " + zone)
	}

	return fqdn[0 : len(fqdn)-len(zone)-1]
}

func (p *DNSVultr) vultrCall(method, endpoint string, args url.Values) []byte {
	url := "https://api.vultr.com" + endpoint
	var reqBody io.Reader

	switch method {
	case "GET":
		url = url + "?" + args.Encode()
	case "POST":
		reqBody = strings.NewReader(args.Encode())
	default:
		panic("Bad HTTP method: " + method)
	}

	req, err := http.NewRequest(method, url, reqBody)
	checkErr(err)
	req.Header.Add("API-Key", p.apiKey)

	if req.Method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := p.client.Do(req)
	checkErr(err)
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	if resp.StatusCode != 200 {
		checkErr(fmt.Errorf("Unexpected '%s' for request: %s\n%s", resp.Status, req.URL.String(), string(data)))
	}

	return data
}

func (p *DNSVultr) GetRecords(zone string) []dnsRecord {
	data := p.vultrCall("GET", "/v1/dns/records", url.Values{
		"domain": {zone}})

	var vultrRecords []vultrRecord
	err := json.Unmarshal(data, &vultrRecords)
	checkErr(err)

	var records []dnsRecord
	for _, vultrRec := range vultrRecords {
		records = append(records, dnsRecord{
			Name:       vultrRec.Name + "." + zone,
			Type:       vultrRec.Type,
			Value:      vultrRec.Data,
			TTL:        vultrRec.TTL,
			ProviderId: vultrRec.RECORDID})
	}

	return records
}

func (p *DNSVultr) CreateRecord(zone string, rec dnsRecord) error {
	p.vultrCall("POST", "/v1/dns/create_record", url.Values{
		"domain": {zone},
		"name":   {stripZone(rec.Name, zone)},
		"type":   {rec.Type},
		"data":   {rec.Value},
		"ttl":    {strconv.Itoa(rec.TTL)}})

	return nil
}

func (p *DNSVultr) UpdateRecord(zone string, rec dnsRecord) error {
	p.vultrCall("POST", "/v1/dns/update_record", url.Values{
		"domain":   {zone},
		"name":     {stripZone(rec.Name, zone)},
		"type":     {rec.Type},
		"data":     {rec.Value},
		"ttl":      {strconv.Itoa(rec.TTL)},
		"RECORDID": {strconv.Itoa(rec.ProviderId)}})

	return nil
}

func (p *DNSVultr) DeleteRecord(zone string, rec dnsRecord) error {
	p.vultrCall("POST", "/v1/dns/delete_record", url.Values{
		"domain":   {zone},
		"RECORDID": {strconv.Itoa(rec.ProviderId)}})

	return nil
}
