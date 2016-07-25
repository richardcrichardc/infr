package main

import (
	"fmt"
	"strings"
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
		case "add":
			backupAddCmd(parseFlags(args, hostsAddFlags))
		case "remove":
			backupRemoveCmd(parseFlags(args, hostsAddFlags))
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

func backupAddCmd(args []string) {
	if len(args) != 2 {
		errorExit("Wrong number of arguments for 'backups add <from host> <to host>")
	}

	from := args[0]
	to := args[1]

	for _, b := range config.Backups {
		if b.From == from && b.To == to {
			errorExit("A backup from '%s' to '%s' is already configured.", from, to)
		}
	}

	fromHost := findHost(from)
	toHost := findHost(to)

	b := backup{From: from, To: to}
	config.Backups = append(config.Backups, &b)
	saveConfig()

	toHost.setupBackupOf(fromHost)
}

func backupRemoveCmd(args []string) {
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
}

func (h *host) setupBackupOf(target *host) {
	backupData := setupBackupAgentData{
		Target:     target.Name,
		InfrDomain: infrDomain(),
	}

	h.SudoScript(setupBackupAgentScript, backupData)

	pubkey := h.RunCaptureStdout("sudo cat /var/lib/backups/.ssh/id_rsa.pub", true)

	snapshotData := setupBackupSnapshotData{
		Key: strings.TrimSpace(pubkey),
		To:  h.Name,
	}

	target.SudoScript(setupBackupSnapshotsScript, snapshotData)
}

type setupBackupAgentData struct {
	Target     string
	InfrDomain string
}

const setupBackupAgentScript = `
# echo commands and exit on error
set -v -e

if [ ! -e "/var/lib/backups" ]; then
    adduser --system --home /var/lib/backups --ingroup sudo backup_agent
    sudo -u backup_agent ssh-keygen -q -N '' -f /var/lib/backups/.ssh/id_rsa

	cat <<'EOF' > /var/lib/backups/backup-host
#!/bin/bash -ex

if [ ! "$#" -eq 1 ];then
	echo Usage: backup-host host
	exit 1
fi

FROM=$1
TO=$(hostname)
DIR=/var/lib/backups/hosts/$FROM
FQDN=$FROM.$(cat /var/lib/backups/INFR_DOMAIN)
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%S)

if [ ! -d $DIR ];then
	echo $DIR is not a directory
	exit 1
fi

cd $DIR

(
	if ! flock -n 9; then
		echo Backup of $FROM already in progress
		exit 1
	fi

	LAST=$(ls | tail -n 1)

	# create a new remote snapshot
	ssh backup_snapshots@$FQDN -C "cd $TO; sudo btrfs subvolume snapshot -r / $TIMESTAMP"

	# handle errors ourselves
	set +e

	# copy the snapshot to this host
	if [ "$LAST" == "" ]; then
		ssh -C backup_snapshots@$FQDN "cd $TO; sudo btrfs send $TIMESTAMP" | sudo btrfs receive .
	else
		ssh -C backup_snapshots@$FQDN "cd $TO; sudo btrfs send -p $LAST $TIMESTAMP" | sudo btrfs receive .
	fi

	RESULT=$?

	# Exit on error
	set -e

	# btrfs receive does not clean up after itself when an error occurs
	if [ ! $RESULT -eq 0 ]; then
	    sudo btrfs subvolume delete $TIMESTAMP
	    exit
	fi

	# delete all but the last remote snapshot
	ssh backup_snapshots@$FQDN -C "cd $TO; ls -r | tail -n+2 | xargs chronic sudo btrfs subvolume delete"
) 9>.lockfile

EOF
chmod +x /var/lib/backups/backup-host

	cat <<'EOF' > /var/lib/backups/backup-all
#!/bin/bash -e

cd /var/lib/backups/hosts

LOGDIR=$(mktemp -d)
trap "rm -fr $LOGDIR" EXIT

for HOST in *; do
  chronic ../backup-host $HOST &> $LOGDIR/$HOST &
done

wait

cd $LOGDIR
for HOST in *; do
	if [ -s $HOST ]; then
		echo Backup of $HOST failed:
		cat $HOST
		echo
	fi
done

EOF
chmod +x /var/lib/backups/backup-all

	cat <<'EOF' > /etc/cron.d/infr-backup
SHELL=/bin/sh
PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin
MAILTO=root

*/5 * * * * backup_agent /var/lib/backups/backup-all
EOF

fi

echo "{{.InfrDomain}}" > /var/lib/backups/INFR_DOMAIN
mkdir -p /var/lib/backups/hosts/{{.Target}}
mkdir -p /var/lib/backups/logs
chown backup_agent:nogroup /var/lib/backups/hosts/{{.Target}} /var/lib/backups/logs


`

type setupBackupSnapshotData struct {
	To  string
	Key string
}

const setupBackupSnapshotsScript = `
# echo commands and exit on error
set -v -e

if [ ! -e "/var/lib/snapshots" ]; then
    adduser --system --home /var/lib/snapshots --shell /bin/bash --ingroup sudo backup_snapshots
    mkdir -p /var/lib/snapshots/.ssh
    touch /var/lib/snapshots/.ssh/authorized_keys
    chown -R backup_snapshots /var/lib/snapshots/.ssh
    chmod 0400 /var/lib/snapshots/.ssh/authorized_keys
fi

if ! grep -Fxq "{{.Key}}" /var/lib/snapshots/.ssh/authorized_keys; then
	echo "{{.Key}}" >> /var/lib/snapshots/.ssh/authorized_keys
fi
mkdir -p /var/lib/snapshots/{{.To}}

`
