package main

import (
	"fmt"
	"infr/rage4"
	"strings"
)

type recordStatus int

const (
	UNKNOWN = iota
	NEW
	CORRECT
	INCORRECT
	EXTRA
)

type dnsRecord struct {
	Name   string
	Type   string
	Value  string
	TTL    int
	Reason string
	Status recordStatus
}

func dnsListCmd(args []string) {
	records := dnsRecordsNeeded()
	managed := dnsIsManaged()

	if managed {
		records = checkDnsRecords(records)
	}

	fmt.Println("FQDN                                     TYPE  VALUE           TTL     FOR             STATUS")
	fmt.Println("=============================================================================================")
	for _, record := range records {
		fmt.Printf("%-40s %-5s %-15s %-7d %-15s %-6s\n",
			record.Name,
			record.Type,
			record.Value,
			record.TTL,
			record.Reason,
			recordStatusString(record.Status))
	}

	if !managed {
		fmt.Printf("\nDNS records are not automatically managed, set 'dns/rage4/..' config settings to enable.\n")
	}

}

func needInfrDomain() string {
	dnsDomain := needGeneralConfig("dns/domain")
	dnsPrefix := needGeneralConfig("dns/prefix")
	return dnsPrefix + "." + dnsDomain
}

func dnsIsManaged() bool {
	return generalConfig("dns/rage4/account") != ""
}

func dnsRecordsNeeded() []dnsRecord {
	infrDomain := needInfrDomain()

	var hosts []host
	loadConfig("hosts", &hosts)

	var records []dnsRecord

	for _, host := range hosts {
		records = append(records, dnsRecord{
			Name:   host.Name + "." + infrDomain,
			Type:   "A",
			Value:  host.PublicIPv4,
			TTL:    3600,
			Reason: "HOST PUBLIC IP"})
	}

	return records
}

func recordStatusString(v recordStatus) string {
	switch v {
	case UNKNOWN:
		return "???"
	case NEW:
		return "NEW"
	case CORRECT:
		return "CORRECT"
	case INCORRECT:
		return "INCORRECT"
	case EXTRA:
		return "EXTRA"
	default:
		panic("Invalid recordStatus")
	}
}

func checkDnsRecords(records []dnsRecord) []dnsRecord {
	dnsDomain := needGeneralConfig("dns/domain")
	infrDomain := needInfrDomain()

	client := rage4.NewClient(needGeneralConfig("dns/rage4/account"), needGeneralConfig("dns/rage4/key"))

	domain, err := client.GetDomainByName(dnsDomain)
	checkErr(err)

	actualRecords, err := client.GetRecords(domain.Id)
	var extras []rage4.Record

	for i, _ := range records {
		records[i].Status = NEW
	}

aRecLoop:
	for _, aRec := range actualRecords {
		if strings.HasSuffix(aRec.Name, infrDomain) {
			for i, rec := range records {
				if rec.Name == aRec.Name {
					if aRec.Type == rec.Type && aRec.Content == rec.Value && aRec.TTL == rec.TTL {
						records[i].Status = CORRECT
					} else {
						records[i].Status = INCORRECT
					}
					continue aRecLoop
				}
			}
			extras = append(extras, aRec)
		}
	}

	for _, extra := range extras {
		records = append(records, dnsRecord{
			Name:   extra.Name,
			Type:   extra.Type,
			Value:  extra.Content,
			TTL:    extra.TTL,
			Reason: "",
			Status: EXTRA})
	}
	return records
}

func dnsCmd(args []string) {
	fmt.Println("DNS")

	client := rage4.NewClient("accounts@tawherotech.nz", "3bc988f9d09d8f316ac71c22f139f732")

	domain, err := client.GetDomainByName("tawherotech.nz")
	checkErr(err)
	fmt.Printf("%#v\n", domain)

	records, err := client.GetRecords(domain.Id)
	checkErr(err)
	for _, rec := range records {
		fmt.Printf("%#v\n", rec)
	}

	usage, err := client.ShowCurrentUsage(domain.Id)
	checkErr(err)
	for _, rec := range usage {
		fmt.Printf("%#v\n", rec)
	}
}

func checkErr(err error) {
	if err != nil {
		errorExit("Error managing DNS: %s", err)
	}
}
