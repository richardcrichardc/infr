package main

import (
	"errors"
	"infr/rage4"
)

type dnsRage4 struct {
	client *rage4.Client

	// cache of last call to getDomain
	lastDomainName string
	lastDomainId   int
}

func newDNSRage4() dnsProvider {
	var p = &dnsRage4{}
	p.client = rage4.NewClient(needGeneralConfig("dnsRage4Account"), needGeneralConfig("dnsRage4Key"))
	return p
}

func (p *dnsRage4) getDomainId(zone string) (int, error) {
	if zone != p.lastDomainName {

		domain, err := p.client.GetDomainByName(zone)
		if err != nil {
			return 0, err
		}

		p.lastDomainName = zone
		p.lastDomainId = domain.Id
	}

	return p.lastDomainId, nil

}

func (p *dnsRage4) GetRecords(zone string) []dnsRecord {
	domainId, err := p.getDomainId(zone)
	checkErr(err)

	rage4Records, err := p.client.GetRecords(domainId)
	if err != nil {
		errorExit("Error quering Rage4 for DNS records: %s", err)
	}

	var records []dnsRecord

	for _, r4Rec := range rage4Records {
		records = append(records, dnsRecord{
			Name:       r4Rec.Name,
			Type:       r4Rec.Type,
			Value:      r4Rec.Content,
			TTL:        r4Rec.TTL,
			ProviderId: r4Rec.Id})
	}

	return records
}

func (p *dnsRage4) CreateRecord(zone string, rec dnsRecord) error {
	domainId, err := p.getDomainId(zone)
	checkErr(err)

	status, err := p.client.CreateRecord(domainId, rage4.Record{
		Name:     rec.Name,
		Content:  rec.Value,
		Type:     rec.Type,
		TTL:      rec.TTL,
		Priority: 1,
		IsActive: true})

	if err != nil {
		return err
	}

	if status.Status == false {
		return errors.New(status.Error)
	}

	return nil

}

func (p *dnsRage4) UpdateRecord(zone string, rec dnsRecord) error {
	status, err := p.client.UpdateRecord(rec.ProviderId, rage4.Record{
		Name:     rec.Name,
		Content:  rec.Value,
		Type:     rec.Type,
		TTL:      rec.TTL,
		Priority: 1,
		IsActive: true})

	if err != nil {
		return err
	}

	if status.Status == false {
		return errors.New(status.Error)
	}

	return nil
}

func (p *dnsRage4) DeleteRecord(zone string, rec dnsRecord) error {
	status, err := p.client.DeleteRecord(rec.ProviderId)

	if err != nil {
		return err
	}

	if status.Status == false {
		return errors.New(status.Error)
	}

	return nil
}
