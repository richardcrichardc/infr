package main

import (
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
	Name       string
	Type       string
	Value      string
	TTL        int
	Reason     string
	Status     recordStatus
	ProviderId int
}

type dnsProvider interface {
	GetRecords(zone string) []dnsRecord
	CreateRecord(zone string, rec dnsRecord) error
	UpdateRecord(zone string, rec dnsRecord) error
	DeleteRecord(zone string, rec dnsRecord) error
}

func infrDomain() string {
	dnsDomain := needGeneralConfig("dnsDomain")
	dnsPrefix := needGeneralConfig("dnsPrefix")
	return dnsPrefix + "." + dnsDomain
}

func vnetDomain() string {
	dnsDomain := needGeneralConfig("dnsDomain")
	vnetPrefix := needGeneralConfig("vnetPrefix")
	return vnetPrefix + "." + dnsDomain
}

func getDnsProvider() dnsProvider {
	switch generalConfig("dnsProvider") {
	case "rage4":
		return newDNSRage4()
	case "vultr":
		return newDNSVultr()
	default:
		return nil
	}
}

func dnsFix() {
	provider := getDnsProvider()
	records := dnsRecordsNeeded()
	records = checkDnsRecords(provider, records)
	fixDnsRecords(provider, records)
}

func dnsRecordsNeeded() []dnsRecord {
	var records []dnsRecord

	for _, host := range config.Hosts {
		records = append(records, dnsRecord{
			Name:   host.FQDN(),
			Type:   "A",
			Value:  host.PublicIPv4,
			TTL:    3600,
			Reason: "HOST PUBLIC IP"})

		records = append(records, dnsRecord{
			Name:   host.VnetFQDN(),
			Type:   "A",
			Value:  host.PrivateIPv4,
			TTL:    3600,
			Reason: "HOST PRIVATE IP"})
	}

	for _, lxc := range config.Lxcs {

		host := lxc.FindHost()

		records = append(records, dnsRecord{
			Name:   lxc.FQDN(),
			Type:   "A",
			Value:  host.PublicIPv4,
			TTL:    3600,
			Reason: "LXC HOST PUBLIC IP"})

		records = append(records, dnsRecord{
			Name:   lxc.VnetFQDN(),
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

func checkDnsRecords(provider dnsProvider, records []dnsRecord) []dnsRecord {
	dnsDomain := needGeneralConfig("dnsDomain")

	actualRecords := provider.GetRecords(dnsDomain)

	var extras []dnsRecord

	for i, _ := range records {
		records[i].Status = MISSING
	}

aRecLoop:
	for _, aRec := range actualRecords {
		if strings.HasSuffix(aRec.Name, infrDomain()) || strings.HasSuffix(aRec.Name, vnetDomain()) {
			for i, rec := range records {
				if rec.Name == aRec.Name {
					records[i].ProviderId = aRec.ProviderId

					if aRec.Type == rec.Type && aRec.Value == rec.Value && aRec.TTL == rec.TTL {
						records[i].Status = CORRECT
					} else {
						records[i].Status = INCORRECT
					}
					continue aRecLoop
				}
			}
			extra := aRec
			extra.Status = EXTRA
			extras = append(extras, extra)
		}
	}

	records = append(records, extras...)

	return records
}

func fixDnsRecords(provider dnsProvider, records []dnsRecord) {
	dnsDomain := needGeneralConfig("dnsDomain")

	for _, rec := range records {
		var err error

		switch rec.Status {
		case MISSING:
			println("Creating DNS record:", rec.Name)
			err = provider.CreateRecord(dnsDomain, rec)
		case CORRECT:
			// do nothing
			continue
		case INCORRECT:
			println("Updating DNS record:", rec.Name)
			err = provider.UpdateRecord(dnsDomain, rec)
		case EXTRA:
			println("Removing DNS record:", rec.Name)
			err = provider.DeleteRecord(dnsDomain, rec)
		default:
			errorExit("Invalid recordStatus: %s %d", rec.Name, rec.Status)
		}

		checkErr(err)
	}
}

func checkErr(err error) {
	if err != nil {
		errorExit("Error managing DNS: %s", err)
	}
}
