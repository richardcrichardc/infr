package main

func files(name string) string {
	file, ok := filesMap[name]
	if !ok {
		panic("No file: " + name)
	}
	return file
}

var filesMap = map[string]string {
    "backup-all": `#!/bin/bash -e
#
# Backup all hosts which have a directory in the top level backup directory
# Backups are performed in paralled and any (error) output is combined for reporting
# This script is periodically run by a cron job

cd /var/lib/backups/backups

LOGDIR=$(mktemp -d)
trap "rm -fr $LOGDIR" EXIT

for HOST in ` + "`" + `compgen -G '*'` + "`" + `; do
  chronic backup-host $HOST &> $LOGDIR/$HOST &
done

wait

cd $LOGDIR
for HOST in ` + "`" + `compgen -G '*'` + "`" + `; do
	if [ -s $HOST ]; then
		echo Backup of $HOST failed:
		cat $HOST
		echo
	fi
done
`,

    "backup-cull-snaps": `#!/bin/bash -ex


cd snapshots-for/$1

# Remove all but last set of backups
for DIR in ` + "`" + `ls -r | tail -n+2` + "`" + `; do
    if [ -n "` + "`" + `ls $DIR` + "`" + `" ]; then
        chronic sudo btrfs subvolume delete $DIR/*
    fi
    rmdir $DIR
done

`,

    "backup-host": `#!/bin/bash -ex
# Perform a backup of remote host by:
#  * sshing in,
#  * taking a btrfs snapshot (for host and each LXC)
#  * pipeing snapshots back using btrfs send
#  * saving snapshot locally using btrfs receive
#  * removing earlier snapshots
#
# Various assumptions are made about where snapshots etc are kept
# other scripts set up \directories and ssh access required for this script to work

if [ ! "$#" -eq 1 ];then
	echo Usage: backup-host host
	exit 1
fi

FROM=$1
TO=$(hostname)
DIR=/var/lib/backups/backups/$FROM
FQDN=$FROM.$(cat /etc/infr-domain)
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

	# clean up any mess left over from last time
	if [ -e incoming ]; then
		if [ -n "` + "`" + `ls incoming` + "`" + `" ]; then
			sudo btrfs subvolume delete incoming/*
		fi
		rmdir incoming
	fi

	LAST=$(ls | tail -n 1)

	mkdir incoming
	cd incoming

	# create then send new remote snapshots
	for SNAP in ` + "`" + `ssh backup_user@$FQDN -C "cd snapshots-for; backup-snap $TO/$TIMESTAMP"` + "`" + `; do
		echo snap $SNAP

		# did we get a successful snapshot for SNAP from last backup
		LAST_SNAP=$LAST/$SNAP
		echo last $LAST_SNAP
		if [ ! -e "../$LAST_SNAP" ]; then
			LAST_SNAP=""
		fi

		# copy the snapshot to this host
		ssh -C backup_user@$FQDN "cd snapshots-for/$TO; sudo backup-send $TIMESTAMP/$SNAP $LAST_SNAP" | sudo btrfs receive .
	done

	# if we get this far, we have successfully transfered all backups, do atomic mv to 'commit' incoming backup
	cd ..
	mv incoming $TIMESTAMP

	# delete all but the last set of remote snapshots
	ssh backup_user@$FQDN -C "backup-cull-snaps $TO"

	# delete old backups
	for DIR in ` + "`" + `ls | backups-to-cull` + "`" + `; do
		if [ -n "` + "`" + `ls $DIR` + "`" + `" ]; then
			chronic sudo btrfs subvolume delete $DIR/*
		fi
		rmdir $DIR
	done



) 9>.lockfile
`,

    "backup-send": `#!/bin/bash
# A wrapper around btrfs send
# Performs an incremental send if both from-snapshot and to-snapshot are presend
# otherwise perform a full send of to-snapshot

SNAPSHOT=$1
PREV=$2

if [ ! -e "$PREV" ]; then
	btrfs send $SNAPSHOT | cat
else
	btrfs send -p $PREV $SNAPSHOT | cat
fi`,

    "backup-snap": `#!/bin/bash -e

mkdir -p $1
cd $1

# snapshot host
sudo btrfs subvolume snapshot -r / host > /dev/null

# snapshot root filesystem of each LXC
for LXC in ` + "`" + `sudo lxc-ls` + "`" + `; do
    sudo btrfs subvolume snapshot -r /var/lib/lxc/$LXC/rootfs lxc-$LXC > /dev/null
done

# send a list of snapshots, so the backup server knows what snapshots to pull
ls
`,

    "backups-to-cull": `#!/usr/bin/python3

# Determine what backups to keep
#  * All in the last 2 hours
#  * Hourlies for the last 2 days
#  * Dailies for the last 2 weeks
#  * Weeklies for the last 12 weeks
#
# Pipe dates into stdin, dates to remove are written to stdout

import sys
from datetime import datetime, timedelta

date_format = "%Y-%m-%dT%H:%M:%S"


def testDates():
    dates = [datetime.now() - timedelta(minutes=m) for m in range(0, 2000, 5)] +\
        [datetime.now() - timedelta(days=d) for d in range(1, 200)]

    return [d.strftime(date_format) for d in dates]


def parseDates(dateStrings):
    dates = []

    for s in dateStrings:
        try:
            dates.append(datetime.strptime(s.strip(), date_format))
        except ValueError:
            print("Invalid date: %s" % s, file=sys.stderr)
            exit(1)

    return dates


def printDates(dates):
    for d in dates:
        print(d.strftime(date_format))


def keep_interval(dates, latest, interval, far_back):
    keep = []
    no_diff = timedelta(seconds=0)

    ideal = latest.replace(minute=0, second=0)
    earliest = ideal - far_back
    while ideal > earliest:
        best = None
        best_diff = timedelta(days=1000000)
        for d in dates:
            diff = d - ideal
            if diff > no_diff and diff < best_diff:
                best = d
                best_diff = diff
        keep.append(best)
        ideal -= interval

    return keep


def to_keep(dates):
    keep = set()

    if dates == []:
        return keep

    # Retension is based on the latest backup time rather than current time
    dates = sorted(dates)
    latest = dates[-1]
    earliest = dates[-1]

    # Keep all within last 2 hours
    hour_ago = latest - timedelta(hours=2)
    keep.update([d for d in dates if d > hour_ago])

    # Keep hourlies for last 2 days
    keep.update(keep_interval(dates, latest.replace(minute=0, second=0), timedelta(hours=1), timedelta(days=2)))

    # Keep dailies for last 2 weeks
    keep.update(keep_interval(dates, latest.replace(hour=0, minute=0, second=0), timedelta(days=1), timedelta(days=14)))

    # Keep weeklies for last 12 weeks
    week_starts = latest.replace(hour=0, minute=0, second=0) - timedelta(days=latest.weekday())
    keep.update(keep_interval(dates, week_starts, timedelta(days=7), timedelta(days=7*12)))

    return keep

if __name__ == "__main__":
    dates = parseDates(sys.stdin.readlines())
    cull = sorted(set(dates) - to_keep(dates))
    printDates(cull)
`,

    "confedit": `#!/usr/bin/python3

import sys
from collections import OrderedDict


def usage():
    print("""Usage: confscriptedit <dest-file>

Config file editor merges the config script provided on stdin into <dest-file>,
the result is saved to <dest-file>.

Config scripts consist of:
# comments

# ^^^ empty lines ^^^^, and
key = value-pairs

Merging occurs by replacing key value-pairs, matching by key, in <dest-file>
then appending all remaining items. Items specified with a blank value are
removed from <dest-file>. All other lines in <dest-file> are copied as is.
""")
    exit(1)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        usage()

    try:
        # read in destfile
        dest = sys.argv[1]
        f = open(dest)
        lines = f.read().splitlines()
        f.close()

        # read stdin into dict
        changes = OrderedDict()
        for line in sys.stdin.read().splitlines():
            if line.strip() == "" or line[0] == "#":
                continue

            key, sep, value = line.partition("=")
            if sep == "":
                print("Cannot understand input:", key)
                exit(1)

            changes[key.strip()] = value.strip()

        # write out original file making substitutions
        changed = set()
        f = open(dest, "w")
        for line in lines:
            key, sep, value = line.partition("=")
            stripped_key = key.strip()
            if sep == "" or (key and key[0] == "#") or stripped_key not in changes:
                f.write("%s\n" % line)
            else:
                f.write("%s = %s\n" % (stripped_key, changes[stripped_key]))
                changed.add(stripped_key)

        # write out new config items
        for key, value in changes.items():
            if key not in changed:
                f.write("%s = %s\n" % (key, value))

        f.close()
        exit(0)

    except IOError as e:
        print(e)
        exit(1)

`,

    "configure.sh": `# Exit on error
set -e

# Leave zerotier networks before joining so interface comes up as zt0
zerotier-cli listnetworks | cut -d ' ' -f 3 | while read networkId; do
   if [ "$networkId" != "<nwid>" ]; then
      zerotier-cli leave $networkId
   fi
done

# Join zerotier network
if [ -n "{{.ZerotierNetworkId}}" ]
then
	zerotier-cli join {{.ZerotierNetworkId}}
fi

# reload haproxy before getting new certificates so .well-known/acme-challenge is enabled for the new domain
service haproxy reload
issue-ssl-certs {{.AdminEmail}}
install-ssl-certs

# reload haproxy after getting new certificates so certificate is used
service haproxy reload
`,

    "help": `.\" generated with Ronn/v0.7.3
.\" http://github.com/rtomayko/ronn/tree/0.7.3
.
.TH "INFR" "" "September 2016" "" ""
.
.SH "NAME"
\fBinfr\fR \- manage virtual hosting infrastructure
.
.SH "USAGE"
\fBinfr\fR [\fIoptions\fR] \fBhosts\fR [\fBlist\fR]
.
.br
\fBinfr\fR [\fIoptions\fR] \fBhosts add\fR [\fB\-p\fR \fIroot\-password\fR] \fIname\fR \fItarget\-IP\-address\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBhost\fR \fIname\fR \fBreconfigure\fR [\fB\-n\fR] [\fB\-s\fR] [\fB\-k\fR] [\fB\-R\fR]
.
.br
\fBinfr\fR [\fIoptions\fR] \fBhost\fR \fIname\fR \fBremove\fR
.
.br
.
.P
\fBinfr\fR [\fIoptions\fR] \fBlxcs\fR \fIhost\fR [\fBlist\fR]
.
.br
\fBinfr\fR [\fIoptions\fR] \fBlxcs\fR \fIhost\fR \fBadd\fR \fIname\fR \fIdistribution\fR \fIrelease\fR \fIhost\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBlxc\fR \fIname\fR \fBshow\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBlxc\fR \fIname\fR \fBadd\-alias\fR \fIFQDN\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBlxc\fR \fIname\fR \fBremove\-alias\fR \fIFQDN\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBlxc\fR \fIname\fR \fBhttp\fR \fBNONE\fR|\fBFORWARD\fR|\fBREDIRECT\-HTTPS\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBlxc\fR \fIname\fR \fBhttps\fR \fBNONE\fR|\fBFORWARD\fR|\fBTERMINATE\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBlxc\fR \fIname\fR \fBhttp\-port\fR \fIport\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBlxc\fR \fIname\fR \fBhttps\-port\fR \fIport\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBlxc\fR \fIname\fR \fBremove\fR
.
.br
.
.P
\fBinfr\fR [\fIoptions\fR] \fBbackups\fR [\fBlist\fR]
.
.br
\fBinfr\fR [\fIoptions\fR] \fBbackups start\fR \fIfrom\-host\fR \fIto\-host\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBbackups stop\fR \fIfrom\-host\fR \fIto\-host\fR
.
.br
.
.P
\fBinfr\fR [\fIoptions\fR] \fBconfig\fR [\fBshow\fR]
.
.br
\fBinfr\fR [\fIoptions\fR] \fBconfig set\fR \fIname\fR \fIvalue\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBconfig unset\fR \fIname\fR
.
.br
.
.P
\fBinfr\fR [\fIoptions\fR] \fBkeys\fR [\fBlist\fR]
.
.br
\fBinfr\fR [\fIoptions\fR] \fBkeys add\fR \fIkeyfile\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBkeys remove\fR] \fIkeyfile\fR
.
.br
.
.P
\fBinfr\fR [\fIoptions\fR] \fBdns\fR [\fBlist\fR]
.
.br
\fBinfr\fR [\fIoptions\fR] \fBdns fix\fR
.
.br
.
.P
\fBinfr\fR [\fIoptions\fR] \fBhelp\fR
.
.br
.
.SH "SYNOPSIS"
\fBname\fR [\fIoptional\fR\.\.\.] \fIflags\fR
`,

    "infr-backup": `SHELL=/bin/sh
PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin
MAILTO=root

*/5 * * * * backup_user backup-all
`,

    "install-software.sh": `# echo commands and exit on error
set -x -e

# stop apt-get prompting for input
export DEBIAN_FRONTEND=noninteractive

# enable backports so we can install certbot
echo "deb http://ftp.debian.org/debian jessie-backports main" > /etc/apt/sources.list.d/backports.list
apt-get update

# remove exim
apt-get -y purge exim4 exim4-base exim4-config exim4-daemon-light

# remove portmap
apt-get -y purge rpcbind

# install various packages
apt-get -y install lxc bridge-utils haproxy ssl-cert webfs btrfs-tools moreutils nullmailer
apt-get -y install certbot -t jessie-backports

# create ssl directory for haproxy
mkdir -p /etc/haproxy/ssl

# create doc_root and .wellknown for certbot
mkdir -p /etc/haproxy/certbot/.well-known

# the certbot package has a cron job to renew certificates on a daily basis
# here we add a daily cron job to install the renewed certificates
ln -sf /usr/local/bin/install-ssl-certs /etc/cron.daily/install-ssl-certs

# enable IP forwarding so nat from private network works
cat <<'EOF' | confedit /etc/sysctl.conf
net.ipv4.ip_forward = 1
EOF
sysctl --system

# install zerotier one
wget -O - https://install.zerotier.com/ | bash

# create backup user and directories
if ! grep -q backup_user /etc/passwd; then
	adduser --system --home /var/lib/backups --no-create-home --shell /bin/bash --ingroup sudo backup_user
fi

mkdir -p /var/lib/backups
mkdir -p /var/lib/backups/backups
mkdir -p /var/lib/backups/archive
mkdir -p /var/lib/backups/snapshots-for
chown backup_user:nogroup /var/lib/backups /var/lib/backups/backups /var/lib/backups/snapshots-for

if [ ! -e "/var/lib/backups/.ssh/id_rsa" ]; then
	sudo -u backup_user ssh-keygen -q -N '' -f /var/lib/backups/.ssh/id_rsa
fi

# setup minimal SSL configuration to avoid chicken and egg problem between haproxy and letencrypt
cat /etc/ssl/certs/ssl-cert-snakeoil.pem /etc/ssl/private/ssl-cert-snakeoil.key > /etc/haproxy/ssl/default.crt
touch /etc/haproxy/ssl-crt-list
`,

    "install-ssl-certs": `#!/bin/bash
# script for installing certs issued by certbot

# remove old certs and cert list
rm -f /etc/haproxy/ssl/*
truncate --size=0 /etc/haproxy/ssl-crt-list

# create default file used when HOST does not match any other certs
cat /etc/ssl/certs/ssl-cert-snakeoil.pem /etc/ssl/private/ssl-cert-snakeoil.key > /etc/haproxy/ssl/default.crt

cat /etc/haproxy/https-domains | while read FQDN; do
  if [ "$FQDN" != "" ]; then
    LIVEDIR=/etc/letsencrypt/live/$FQDN
    if [ -e "$LIVEDIR" ]; then
        CERTFILE=/etc/haproxy/ssl/$FQDN.crt
        echo $CERTFILE >> /etc/haproxy/ssl-crt-list
        cat $LIVEDIR/fullchain.pem $LIVEDIR/privkey.pem  > $CERTFILE
    fi
  fi
done
`,

    "interfaces": `# AUTOMATICALLY GENERATED - DO NOT EDIT

# This file describes the network interfaces available on your system
# and how to activate them. For more information, see interfaces(5).

source /etc/network/interfaces.d/*

# The loopback network interface
auto lo
iface lo inet loopback

# The primary network interface
auto eth0
allow-hotplug eth0
iface eth0 inet dhcp

auto br0
iface br0 inet static
    bridge_ports none
    address {{.PrivateIPv4}}
    netmask {{.PrivateNetworkMask}}
    up iptables -t nat -A POSTROUTING -s {{.PrivateNetwork}} -o eth0 -j MASQUERADE
    down iptables -t nat -D POSTROUTING -s {{.PrivateNetwork}} -o eth0 -j MASQUERADE

auto zt0
allow-hotplug zt0
iface zt0 inet manual
    up brctl addif br0 zt0
    down brctl delif br0 zt0
`,

    "issue-ssl-certs": `#!/bin/bash
# script for getting certbot to issue ssl certificates

cat /etc/haproxy/https-domains | while read FQDN; do
  if [ "$FQDN" != "" ]; then
  	certbot certonly --webroot --quiet --keep --agree-tos --webroot-path /etc/haproxy/certbot --email $1 -d $FQDN
  fi
done
`,

    "lock-host": `#!/usr/bin/python2

import fcntl, time

f = open("/tmp/infr-host-lock", "w")
try:
    fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)
except IOError:
    print("ALREADY LOCKED")
    exit(1)

print("LOCKED")

while True:
    time.sleep(365*24*3600)
`,

    "no-backend.http": `HTTP/1.0 404 Service Unavailable
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>404 Not Found</h1>
No such site.
</body></html>
`,

    "webfsd.conf": `web_root="/etc/haproxy/certbot"
web_ip="127.0.0.1"
web_port="9980"
web_user="www-data"
web_group="www-data"
web_extras="-j"
`,

}
