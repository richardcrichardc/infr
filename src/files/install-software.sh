# echo commands and exit on error
set -x -e

# stop apt-get prompting for input
export DEBIAN_FRONTEND=noninteractive

# enable backports so we can install certbot
echo "deb http://ftp.debian.org/debian jessie-backports main" > /etc/apt/sources.list.d/backports.list
apt-get update

# remove exim
apt-get -y purge exim4 exim4-base exim4-config exim4-daemon-light

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
