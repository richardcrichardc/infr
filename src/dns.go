package main

import (
	"fmt"
	"infr/rage4"
	"strings"
)

type recordStatus int

const (
	UNKNOWN = iota
	MISSING
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

	fmt.Println("FQDN                                     TYPE  VALUE           TTL     FOR                 STATUS")
	fmt.Println("=================================================================================================")
	for _, record := range records {
		fmt.Printf("%-40s %-5s %-15s %-7d %-19s %-6s\n",
			record.Name,
			record.Type,
			record.Value,
			record.TTL,
			record.Reason,
			recordStatusString(record.Status))
	}

	if !managed {
		fmt.Printf("\nDNS records are not automatically managed, set 'dnsRage4..' config settings to enable.\n")
	}

}

func dnsFixCmd(args []string) {
	dnsFix()
}

func dnsFix() {
	records := dnsRecordsNeeded()
	records = checkDnsRecords(records)
	fixDnsRecords(records)
}

func needInfrDomain() string {
	dnsDomain := needGeneralConfig("dnsDomain")
	dnsPrefix := needGeneralConfig("dnsPrefix")
	return dnsPrefix + "." + dnsDomain
}

func needVnetDomain() string {
	dnsDomain := needGeneralConfig("dnsDomain")
	vnetPrefix := needGeneralConfig("vnetPrefix")
	return vnetPrefix + "." + dnsDomain
}

func dnsIsManaged() bool {
	return generalConfig("dnsRage4Account") != ""
}

func dnsRecordsNeeded() []dnsRecord {
	infrDomain := needInfrDomain()
	vnetDomain := needVnetDomain()

	var records []dnsRecord

	for _, host := range config.Hosts {
		records = append(records, dnsRecord{
			Name:   host.Name + "." + infrDomain,
			Type:   "A",
			Value:  host.PublicIPv4,
			TTL:    3600,
			Reason: "HOST PUBLIC IP"})

		records = append(records, dnsRecord{
			Name:   host.Name + "." + vnetDomain,
			Type:   "A",
			Value:  host.PrivateIPv4,
			TTL:    3600,
			Reason: "HOST PRIVATE IP"})
	}

	for _, lxc := range config.Lxcs {

		host := lxc.FindHost()

		records = append(records, dnsRecord{
			Name:   lxc.Name + "." + infrDomain,
			Type:   "A",
			Value:  host.PublicIPv4,
			TTL:    3600,
			Reason: "LXC HOST PUBLIC IP"})

		records = append(records, dnsRecord{
			Name:   lxc.Name + "." + vnetDomain,
			Type:   "A",
			Value:  lxc.PrivateIPv4,
			TTL:    3600,
			Reason: "LXC PRIVATE IP"})
	}

	return records
}

func recordStatusString(v recordStatus) string {
	switch v {
	case UNKNOWN:
		return "???"
	case MISSING:
		return "MISSING"
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
	dnsDomain := needGeneralConfig("dnsDomain")
	infrDomain := needInfrDomain()
	vnetDomain := needVnetDomain()

	client := rage4.NewClient(needGeneralConfig("dnsRage4Account"), needGeneralConfig("dnsRage4Key"))

	domain, err := client.GetDomainByName(dnsDomain)
	checkErr(err)

	actualRecords, err := client.GetRecords(domain.Id)
	var extras []rage4.Record

	for i, _ := range records {
		records[i].Status = MISSING
	}

aRecLoop:
	for _, aRec := range actualRecords {
		if strings.HasSuffix(aRec.Name, infrDomain) || strings.HasSuffix(aRec.Name, vnetDomain) {
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
	dnsDomain := needGeneralConfig("dnsDomain")

	client := rage4.NewClient(needGeneralConfig("dnsRage4Account"), needGeneralConfig("dnsRage4Key"))

	domain, err := client.GetDomainByName(dnsDomain)
	checkErr(err)

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
		case MISSING:
			println("Creating DNS record:", rec.Name)
			status, err = client.CreateRecord(domain.Id, rage4Rec)
		case CORRECT:
			// do nothing
			continue
		case INCORRECT:
			println("Updating DNS record:", rec.Name)
			status, err = client.UpdateRecord(rec.Rage4Id, rage4Rec)
		case EXTRA:
			println("Removing DNS record:", rec.Name)
			status, err = client.DeleteRecord(rec.Rage4Id)
		default:
			errorExit("Invalid recordStatus: %s %d", rec.Name, rec.Status)
		}

		checkErr(err)

		if status.Status == false {
			errorExit("Error: %s", status.Error)
		}
	}

}

func checkErr(err error) {
	if err != nil {
		errorExit("Error managing DNS: %s", err)
	}
}
