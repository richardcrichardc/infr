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

    "backup-host": `#!/bin/bash -ex
# Perform a backup of remote host by:
#  * sshing in,
#  * taking a btrfs snapshot
#  * pipeing snapshot back using btrfs send
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

	# delete old backups
	ls | backups-to-cull| xargs -r chronic sudo btrfs subvolume delete

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

issue-ssl-certs {{.AdminEmail}}
install-ssl-certs

service haproxy reload
`,

    "help/hosts": `.\" generated with Ronn/v0.7.3
.\" http://github.com/rtomayko/ronn/tree/0.7.3
.
.TH "INFR\-HOSTS" "" "September 2016" "" ""
.
.SH "NAME"
\fBinfr\-hosts\fR \- manage virtual hosting infrastructure
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

`,

    "help/summary": `.\" generated with Ronn/v0.7.3
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
\fBinfr\fR [\fIoptions\fR] \fBlxc\fR \fIname\fR \fBhttp\fR \fBNONE\fR|\fBFORWARD\fR|\fBREDIRECT\fR
.
.br
\fBinfr\fR [\fIoptions\fR] \fBlxc\fR \fIname\fR \fBhttps\fR \fBNONE\fR|\fBFORWARD\fR
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

    "help/tutorial": `.\" generated with Ronn/v0.7.3
.\" http://github.com/rtomayko/ronn/tree/0.7.3
.
.TH "README" "" "September 2016" "" ""
.
.SH "Introduction"
To be done
.
.SH "Getting Started"
.
.SS "Binary Download"
.
.IP "\(bu" 4
Latest x86\-64 Linux binary \fIhttps://tawherotech\.nz/infr/infr\fR
.
.IP "" 0
.
.SS "Building from source"
.
.IP "1." 4
Install Go \fIhttps://golang\.org/doc/install\fR
.
.IP "2." 4
Install Ronn \fIhttps://github\.com/rtomayko/ronn\fR (try \fBapt\-get install ruby\-ronn\fR)
.
.IP "3." 4
Clone this repo: \fBgit clone https://github\.com/richardcrichardc/infr\.git\fR
.
.IP "4." 4
Run the build script: \fBcd infr; \./build\fR
.
.IP "5." 4
Run Infr: \fB\./infr\fR
.
.IP "" 0
.
.SS "Other Prerequisites"
Adding Hosts involves creating a custom build of iPXE \fIhttp://ipxe\.org/\fR so you will need a working C compiler and some libraries on your machine\.
.
.P
If you are running a Debian based distribution you can make sure your have these dependencies by running:
.
.IP "" 4
.
.nf

$ sudo apt\-get install build\-essential liblzma\-dev
.
.fi
.
.IP "" 0
.
.SS "Initial configuration"
Before you can use Infr you need a minimal configuration\. Run \fBinfr init\fR and you will be prompted for the essential settings\.
.
.P
Here are the options you will be prompted for:
.
.IP "\(bu" 4
\fBvnetNetwork\fR (default \fB10\.8\.0\.0/16\fR) \- Hosts and Lxcs have IP addresses automatically assigned on a virtual private network\. This setting determines the range of IP addresses used by the private network\.
.
.IP "\(bu" 4
\fBvnetPoolStart\fR (default \fB10\.8\.1\.0\fR) \- the first address within \fBvnetNetwork\fR that will be allocated\.
.
.IP "\(bu" 4
\fBdnsDomain\fR \- domain name used to compose Host and LXC domain names\. This is used in conjuction with the next wto settings\.
.
.IP "\(bu" 4
\fBdnsPrefix\fR (default \fBinfr\fR) \- each Host and LXC is allocated a domain name for it\'s public IP address\. E\.g\. if the LXC is called \fBbob\fR dnsPrefix is \fBinfr\fR and \fBdnsDomain\fR is \fBmydomain\.com, the LXC will have a domain name of\fRbob\.infr\.mydomain\.com` + "`" + ` allocated\.
.
.IP "\(bu" 4
\fBvnetPrefix\fR (default \fBvnet\fR) \- each Host and LXC is allocated a domain name for it\'s private IP address\. E\.g\. if the LXC is called \fBbob\fR vnetPrefix is \fBvnet\fR and \fBdnsDomain\fR is \fBmydomain\.com, the LXC will have a domain name of\fRbob\.vnet\.mydomain\.com` + "`" + ` allocated\.
.
.IP "\(bu" 4
Initial SSH public key (default \fB$HOME/\.ssh/id_rsa\.pub\fR \- infr maintains a set of SSH keys for everyone who should be able to administrate the Hosts and LXCs\.
.
.IP "\(bu" 4
\fBadminEmail\fR \- Will be removing this as a required initial option\.
.
.IP "" 0
.
.P
Once set up, these options can be changed using the \fBinfr config\fR command\.
.
.SS "Adding your first Host"
Now that you have your minimal configuration, you can add a host\.
.
.P
First you will need a server to use as the Host\. Infr is designed to use VPSs from VPS providers such as Linode, Digital Ocean and Vultr, it can however use any Linux servers on the public internet\.
.
.P
So go to your favourite VPS provider and spin up a host\. It does not matter what distribution you select as we will be replacing it immediately to get a BTRFS filesystem used by the backup\. All you need is to know the host\'s IP address abd have root access via SSH, using the public key you added above, or with a password\. DO NOT USE A MACHINE WITH DATA ON IT YOU WANT TO KEEP, THE ENTIRE HARD DISK WILL BE ERASED\.
.
.P
Once the machine is up you can add it to the machines managed by infr by running the command:
.
.IP "" 4
.
.nf

$ infr hosts add [\-p <root\-password>] <name> <ip\-address>
.
.fi
.
.IP "" 0
.
.P
Where:
.
.IP "\(bu" 4
\fBroot\-password\fR is the the machine\'s root password, this can be skipped if your SSH key is authorised to login as root
.
.IP "\(bu" 4
\fBname\fR is what you want to call the host, and
.
.IP "\(bu" 4
\fBip\-address\fR is the machines public ip address
.
.IP "" 0
.
.P
This will start the bootstrap process which replaces the hosts operating system with Debian 8 on a BTRFS filesystem\. It takes 10\-15 minutes, look \fIhere\fR if you want to know how hosts are installed\.
.
.P
Once your host is set up you can list your hosts:
.
.IP "" 4
.
.nf

$ infr hosts
NAME            PUBLIC IP       PRIVATE IP
==========================================
bob             45\.32\.191\.151   10\.8\.1\.1
.
.fi
.
.IP "" 0
.
.P
We have not configured a DNS Provider yet, so these records have a status of \'???\'\. Let\'s create a LXC first, then we will sort out these DNS records, before SSHing into the Hosts and LXCs\.
.
.SS "Adding your First LXC"
Now that you have a Host you can put an LXC on it:
.
.IP "" 4
.
.nf

$ infr lxcs add fozzie ubuntu xenial bob
.
.fi
.
.IP "" 0
.
.P
That will whir away for a minute or two\. Then you can list the LXCs:
.
.IP "" 4
.
.nf

$ infr lxcs
NAME            HOST            PUBLIC IP       PRIVATE IP
=========================================================
fozzie          kaiiwi          45\.32\.191\.151   10\.8\.1\.2
.
.fi
.
.IP "" 0
.
.P
And find out more about a particular LXC:
.
.IP "" 4
.
.nf

$ \./infr lxc bob
Name:          fozzie
Host:          bob
Distro:        ubuntu xenial
Aliases:
HTTP:          NONE
HTTPS:         NONE
LXC Http Port: 80
TCP Forwards:
.
.fi
.
.IP "" 0
.
.SS "Setting up a DNS Provider"
You now have a Host with an LXC on it\. Infr will manage the canonical domain names if you let it:
.
.IP "" 4
.
.nf

$ infr dns
FQDN                                     TYPE  VALUE           TTL     FOR                 STATUS
=================================================================================================
bob\.infr\.mydomain\.com                    A     45\.32\.191\.151   3600    HOST PUBLIC IP      ???
bob\.vnet\.mydomain\.com                    A     10\.8\.1\.1        3600    HOST PRIVATE IP     ???
fozzie\.infr\.mydomain\.com                 A     45\.32\.191\.151   3600    LXC HOST PUBLIC IP  ???
fozzie\.vnet\.mydomain\.com                 A     10\.8\.1\.2        3600    LXC PRIVATE IP      ???

DNS records are not automatically managed, set \'dnsProvider\' and related config settings to enable\.
.
.fi
.
.IP "" 0
.
.P
These are all the DNS records Infr wants to manage for you\. If you configure a DNS Provider, Infr will talk to the DNS Provider\'s API and configure these for you automatically\.
.
.P
Note they are all subdomains of \fBinfr\.mydomain\.com\fR and \fBvnet\.mydomain\.com\fR which you specified when configuring Infr\. This is important, as you almost definately have other DNS records in your \fBmydomain\.com\fR zone which you don\'t want Infr to mess with\. Infr will not touch any DNS records unless they have one of these two suffixes\. However if you manually add DNS records with one of these suffixes on your DNS providers control panel, Infr will happily blast them away\.
.
.P
Currently Infr supports two DNS Providers, Vultr \fIhttps://www\.vultr\.com/\fR and Rage4 \fIhttps://rage4\.com/\fR\. If you want to use one of these:
.
.IP "1." 4
Log into your DNS Provider, make sure a zone exists for your \fBdnsDomain\fR and get your API keys
.
.IP "2." 4
Run \fBinfr config set dnsProvider vultr\fR or \fBinfr config set dnsProvider rage4\fR to tell Infr which provider to use
.
.IP "3." 4
.
.IP "\(bu" 4
For Vultr you need to use \fBinfr config set\fR to set \fBdnsVultrAPIKey\fR
.
.IP "\(bu" 4
For Rage4 you need to use \fBinfr config set\fR to set \fBdnsRage4Account\fR (to the email address you log into Rage4 with) and \fBdnsRage4Key\fR
.
.IP "" 0

.
.IP "4." 4
Run \fBinfr dns\fR to check Infr can access the API\. The DNS records should now show up with a Status of MISSING\.
.
.IP "5." 4
Run \fBinfr dns fix\fR to tell Infr to create the missing DNS records
.
.IP "" 0
.
.P
We plan to add new DNS Providers in the future, in particular we are likely to add Linode and Digital Ocean\.
.
.P
If you are unable or don\'t want to set up a DNS provider right now, you can simulate the result by copying and pasting from \fBinfr DNS\fR into your \fB/etc/hosts\fR file\. Doing so will make trying out the rest of Infr more straight forward\.
.
.SS "SSHing into your Hosts and LXCs"
Now that you have a Host an LXC and working DNS you can SSH into the machines and have a look around\. Lets SSH into the Host:
.
.IP "" 4
.
.nf

$ ssh \-A manager@bob\.infr\.mydomain\.com
.
.fi
.
.IP "" 0
.
.P
Manager is the shared management account\. It has no password so can only be accessed by users who\'s SSH keys have been added using \fBssh keys add <keyfile>\fR\. Manager is a sudoer with the NOPASSWD option, so you can use sudo to administrate the Host\.
.
.P
Note the \'\-A\' option when SSHing into the Host\. This enables SSH key forwarding, which means you can now SSH into the the LXCs on the host:
.
.IP "" 4
.
.nf

$ ssh fozzie\.vnet\.mydomain\.com
.
.fi
.
.IP "" 0
.
.SH "How it Works"
.
.SS "<a name=\"how\-hosts\-are\-installed\"></a> How Hosts are installed"
.
.IP "1." 4
First Infr will download and build iPXE which is used to start a network install of Debian\. (This will fail if you don\'t have a C compiler set up \- run \fBsudo apt\-get install build\-essential\fR or similar if you have problems\.)
.
.IP "2." 4
Next it will upload the configuration settings for the installer to an anonymous Gist on Github\. (This includes the SSH public key you specified above \- if this worries you review how public key crypto works before running the \fBhosts add\fR command\.)
.
.IP "3." 4
Then Infr will SCP iPXE up to the host\.
.
.IP "4." 4
Next writes to the hosts hard disk will be frozen, iPXE will be written over the bootblock of the hard disk and the host will be rebooted\.
.
.IP "5." 4
.
.IP "\(bu" 4
Sets up BTRFS as the filesystem, which is used for creating backups
.
.IP "\(bu" 4
Installs your SSH public key so the machine can be accesses remotely
.
.IP "" 0

.
.IP "6." 4
Once Debian is installed Infr SSHs in and sets up the host, essentially installing HAProxy and setting it\'s initial configuration\.
.
.IP "" 0

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
    down brctl delif br0 ` + "`" + `zt0
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
