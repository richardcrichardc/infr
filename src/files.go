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

    "backup-host": `#!/bin/bash -ex

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

	LAST=$(ls | tail -n 1)

	# create a new remote snapshot
	ssh backup_user@$FQDN -C "cd snapshots-for; mkdir -p $TO; cd $TO; sudo btrfs subvolume snapshot -r / $TIMESTAMP"

	# handle errors ourselves
	set +e

	# copy the snapshot to this host
	ssh -C backup_user@$FQDN "cd snapshots-for/$TO; sudo backup-send $TIMESTAMP $LAST" | sudo btrfs receive .

	RESULT=$?

	# Exit on error
	set -e

	# btrfs receive does not clean up after itself when an error occurs
	if [ ! $RESULT -eq 0 ]; then
	    sudo btrfs subvolume delete $TIMESTAMP
	    exit
	fi

	# delete all but the last remote snapshot
	ssh backup_user@$FQDN -C "cd snapshots-for/$TO; ls -r | tail -n+2 | xargs -r chronic sudo btrfs subvolume delete"
) 9>.lockfile
`,

    "backup-send": `#!/bin/bash

SNAPSHOT=$1
PREV=$2

if [ ! -e "$PREV" ]; then
	btrfs send $SNAPSHOT | cat
else
	btrfs send -p $PREV $SNAPSHOT | cat
fi`,

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
EOF
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

/etc/haproxy/issue-ssl-certs {{.AdminEmail}}
/etc/haproxy/install-ssl-certs

service haproxy reload
`,

    "infr-backup": `SHELL=/bin/sh
PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin
MAILTO=root

*/5 * * * * backup_user backup-all
`,

    "install-software.sh": `# echo commands and exit on error
set -v -e

# enable backports so we can install certbot

echo "deb http://ftp.debian.org/debian jessie-backports main" > /etc/apt/sources.list.d/backports.list
apt-get update

# install various packages
apt-get -y install lxc bridge-utils haproxy ssl-cert webfs btrfs-tools moreutils
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
`,

    "install-ssl-certs": `#!/bin/bash
# script for installing certs issued by certbot

# remove old certs and cert list
rm -f /etc/haproxy/ssl/*
truncate --size=0 /etc/haproxy/ssl-crt-list

# create default file used when HOST does not match any other certs
cat /etc/ssl/certs/ssl-cert-snakeoil.pem /etc/ssl/private/ssl-cert-snakeoil.key > /etc/haproxy/ssl/default.crt

cat /etc/haproxy/https-domains | while read FQDN; do
  if [ "\$FQDN" != "" ]; then
    LIVEDIR=/etc/letsencrypt/live/\$FQDN
    if [ -e "\$LIVEDIR" ]; then
        CERTFILE=/etc/haproxy/ssl/\$FQDN.crt
        echo \$CERTFILE >> /etc/haproxy/ssl-crt-list
        cat \$LIVEDIR/fullchain.pem \$LIVEDIR/privkey.pem  > \$CERTFILE
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
    down brctl delif br0 ` + "`" + `zt0
`,

    "issue-ssl-certs": `#!/bin/bash
# script for getting certbot to issue ssl certificates

cat /etc/haproxy/https-domains | while read FQDN; do
  if [ "\$FQDN" != "" ]; then
  	certbot certonly --webroot --quiet --keep --agree-tos --webroot-path /etc/haproxy/certbot --email \$1 -d \$FQDN
  fi
done
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
