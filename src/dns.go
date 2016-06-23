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
	Name    string
	Type    string
	Value   string
	TTL     int
	Reason  string
	Status  recordStatus
	Rage4Id int
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

func dnsFixCmd(args []string) {
	records := dnsRecordsNeeded()
	records = checkDnsRecords(records)
	fixDnsRecords(records)
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
					records[i].Rage4Id = aRec.Id

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
			Name:    extra.Name,
			Type:    extra.Type,
			Value:   extra.Content,
			TTL:     extra.TTL,
			Reason:  "",
			Status:  EXTRA,
			Rage4Id: extra.Id})
	}
	return records
}

func fixDnsRecords(records []dnsRecord) {
	dnsDomain := needGeneralConfig("dns/domain")

	client := rage4.NewClient(needGeneralConfig("dns/rage4/account"), needGeneralConfig("dns/rage4/key"))

	domain, err := client.GetDomainByName(dnsDomain)
	checkErr(err)

	// Make it fail
	client = &rage4.Client{}

	for _, rec := range records {

		rage4Rec := rage4.Record{
			Id:       rec.Rage4Id,
			Name:     rec.Name,
			Content:  rec.Value,
			Type:     rec.Type,
			TTL:      rec.TTL,
			Priority: 1,
			DomainId: domain.Id,
			IsActive: true}

		var status rage4.Status

		switch rec.Status {
		case NEW:
			println("Create", rec.Name, rec.Rage4Id)
			status, err = client.CreateRecord(domain.Id, rage4Rec)
		case INCORRECT:
			println("Fix", rec.Name, rec.Rage4Id)
			status, err = client.UpdateRecord(rec.Rage4Id, rage4Rec)
		case EXTRA:
			println("Remove", rec.Name, rec.Rage4Id)
			status, err = client.DeleteRecord(rec.Rage4Id)
		default:
			panic("Invalid recordStatus")
		}

		fmt.Printf("Status: %#v\n", status)
		checkErr(err)
	}

}

func checkErr(err error) {
	if err != nil {
		errorExit("Error managing DNS: %s", err)
	}
}
