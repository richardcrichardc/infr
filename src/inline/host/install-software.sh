# This script is idempotent

# echo commands and exit on error
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


# script for getting certbot to issue ssl certificates
cat << EOF > /etc/haproxy/issue-ssl-certs
#!/bin/bash

cat /etc/haproxy/https-domains | while read FQDN; do
  if [ "\$FQDN" != "" ]; then
  	certbot certonly --webroot --quiet --keep --agree-tos --webroot-path /etc/haproxy/certbot --email \$1 -d \$FQDN
  fi
done
EOF
chmod +x /etc/haproxy/issue-ssl-certs


# script for installing certs issued by certbot
cat << EOF > /etc/haproxy/install-ssl-certs
#!/bin/bash

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
EOF
chmod +x /etc/haproxy/install-ssl-certs


# the certbot package has a cron job to renew certificates on a daily basis
# here we add a daily cron job to install the renewed certificates
ln -sf /etc/haproxy/install-ssl-certs /etc/cron.daily/install-ssl-certs

# config file for webfs - web server used for hosting .web-known directory used to issue ssl certificates
cat << EOF > /etc/webfsd.conf

web_root="/etc/haproxy/certbot"
web_ip="127.0.0.1"
web_port="9980"
web_user="www-data"
web_group="www-data"
web_extras="-j"
EOF

service webfs restart

# install confedit script used by this and other scripts
cat << EOF > /usr/local/bin/confedit
#!/usr/bin/python3

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
chmod +x /usr/local/bin/confedit

# enable IP forwarding so nat from private network works
cat <<'EOF' | confedit /etc/sysctl.conf
net.ipv4.ip_forward = 1
EOF
sysctl --system

# install zerotier one
wget -O - https://install.zerotier.com/ | bash