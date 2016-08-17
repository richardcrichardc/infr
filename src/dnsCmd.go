package main

import (
	"fmt"
)

func dnsCmd(args []string) {
	if len(args) == 0 {
		dnsListCmd(args)
	} else {
		switch args[0] {
		case "list":
			dnsListCmd(parseFlags(args, noFlags))
		case "fix":
			dnsFixCmd(parseFlags(args, noFlags))
		default:
			errorExit("Invalid command: %s", args[0])
		}
	}
}

func dnsListCmd(args []string) {
	records := dnsRecordsNeeded()
	provider := getDnsProvider()

	if provider != nil {
		records = checkDnsRecords(provider, records)
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

	if provider == nil {
		fmt.Printf("\nDNS records are not automatically managed, set 'dnsRage4..' config settings to enable.\n")
	}

}

func dnsFixCmd(args []string) {
	dnsFix()
}
