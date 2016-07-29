package main

import (
	"fmt"
)

type backup struct {
	From string
	To   string
}

func backupsCmd(args []string) {
	if len(args) == 0 {
		backupListCmd(args)
	} else {
		switch args[0] {
		case "list":
			backupListCmd(parseFlags(args, noFlags))
		case "start":
			backupStartCmd(parseFlags(args, hostsAddFlags))
		case "stop":
			backupStopCmd(parseFlags(args, hostsAddFlags))
		default:
			errorExit("Invalid command: backup %s", args[0])
		}
	}
}

func backupListCmd(args []string) {
	if len(args) != 0 {
		errorExit("Too many arguments for 'backups [list]'.")
	}

	fmt.Printf("FROM            TO\n")
	fmt.Printf("================================\n")
	for _, b := range config.Backups {
		fmt.Printf("%-15s %-15s\n", b.From, b.To)
	}
}

func backupStartCmd(args []string) {
	if len(args) != 2 {
		errorExit("Wrong number of arguments for 'backups add <from host> <to host>")
	}

	from := args[0]
	to := args[1]

	fromHost := findHost(from)
	toHost := findHost(to)

	exists := false
	for _, b := range config.Backups {
		if b.From == from && b.To == to {
			exists = true
		}
	}

	if !exists {
		config.Backups = append(config.Backups, &backup{From: from, To: to})
		saveConfig()
	}

	fromHost.startBackupTo(toHost)
}

func backupStopCmd(args []string) {
	if len(args) != 2 {
		errorExit("Wrong number of arguments for 'backups remove <from host> <to host>")
	}

	from := args[0]
	to := args[1]

	var newBackups []*backup

	for _, b := range config.Backups {
		if b.From != from || b.To != to {
			newBackups = append(newBackups, b)
		}
	}

	if len(config.Backups) == len(newBackups) {
		errorExit("Backup from '%s' to '%s' not found.", from, to)
	}

	config.Backups = newBackups
	saveConfig()

	findHost(from).stopBackupTo(findHost(to))
}

func (fromHost *host) startBackupTo(toHost *host) {
	// Make sure backup_user on to-host can ssh into from-host
	pubkey := toHost.SudoCaptureStdout("cat /var/lib/backups/.ssh/id_rsa.pub")
	fromHost.SudoScript(`
if ! grep -Fxq '{{.}}' /var/lib/backups/.ssh/authorized_keys; then
	echo '{{.}}' >> /var/lib/backups/.ssh/authorized_keys
fi`, pubkey)

	// Create (or restore) directory for from-host
	// Scripts installed in host.INstalSoftware will now start performing backups
	toHost.Sudo("cd /var/lib/backups/backups/; mv ../archive/" + fromHost.Name + " . || mkdir -p " + fromHost.Name)
	toHost.Sudo("chown backup_user:nogroup /var/lib/backups/backups/" + fromHost.Name)
}

func (fromHost *host) stopBackupTo(toHost *host) {
	// Move the backup directory on to-host into archive directory
	// Backups will stop being performed
	toHost.Sudo("mv /var/lib/backups/backups/" + fromHost.Name + " /var/lib/backups/archive")

	// Remove scanpshots on from-host
	fromHost.SudoScript(`
cd /var/lib/backups/snapshots-for

if [ -e {{.}} ]; then
	SNAPS=$(compgen -G "{{.}}/*")
	if [ $SNAPS != "" ]; then
		btrfs subvolume delete $SNAPS
	fi

	rmdir {{.}}
fi
`, toHost.Name)

}
