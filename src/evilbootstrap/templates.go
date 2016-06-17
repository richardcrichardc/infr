package evilbootstrap

const ipxeTemplate = `#!ipxe
dhcp
kernel http://http.us.debian.org/debian/dists/jessie/main/installer-amd64/current/images/netboot/debian-installer/amd64/linux
initrd http://http.us.debian.org/debian/dists/jessie/main/installer-amd64/current/images/netboot/debian-installer/amd64/initrd.gz
initrd %s preseed.cfg
boot
`

const hostsTemplate = `127.0.0.1       %s.%s %s localhost

# The following lines are desirable for IPv6 capable hosts
::1     localhost ip6-localhost ip6-loopback
ff02::1 ip6-allnodes
ff02::2 ip6-allrouters
`

const preseedTemplate = `
# Install debian with btrfs root filesystem
# See https://www.debian.org/releases/jessie/amd64/apbs04.html.en for documentation

d-i debian-installer/locale string en_US
d-i keyboard-configuration/xkb-keymap select us

d-i netcfg/choose_interface select auto
d-i netcfg/get_hostname string unassigned-hostname
d-i netcfg/get_domain string unassigned-domain

d-i mirror/country string manual
d-i mirror/http/hostname string http.us.debian.org
d-i mirror/http/directory string /debian
d-i mirror/http/proxy string

d-i passwd/root-login boolean false
d-i passwd/user-fullname string Manager
d-i passwd/username string manager
d-i passwd/user-password password thiswillbedisabled
d-i passwd/user-password-again password thiswillbedisabled

d-i clock-setup/utc boolean true
d-i time/zone string UTC
d-i clock-setup/ntp boolean true

d-i partman-auto/method string regular
d-i partman-auto/expert_recipe string   \
	1000 10000 10000000 btrfs			\
		$primary{ }						\
		$bootable{ }					\
		method{ format }				\
		format{ }						\
		use_filesystem{ }				\
		filesystem{ btrfs }				\
		mountpoint{ / } .				\
										\
	500 50000 100%% linux-swap			\
		method{ swap }					\
		format{ } .						\
d-i partman-partitioning/confirm_write_new_label boolean true
d-i partman/choose_partition select finish
d-i partman/confirm boolean true
d-i partman/confirm_nooverwrite boolean true

tasksel tasksel/first multiselect standard, ssh-server
d-i pkgsel/include string btrfs-tools

popularity-contest popularity-contest/participate boolean false

d-i grub-installer/only_debian boolean true
d-i grub-installer/bootdev  string default

# Add ssh authorised keys
# Disable password
# Allow sudo without password
d-i preseed/late_command string \
	mkdir /target/home/manager/.ssh																		&&\
	chmod u=rwx /target/home/manager/.ssh																&&\
	echo "%s" > /target/home/manager/.ssh/authorized_keys												&&\
	chmod u=rw /target/home/manager/.ssh/authorized_keys												&&\
	sed -i 's/manager:[^:]*:/manager:!:/' /target/etc/shadow											&&\
	sed -i 's/sudo[[:space:]]*ALL=(ALL:ALL) ALL/sudo ALL=(ALL:ALL) NOPASSWD:ALL/' /target/etc/sudoers

d-i finish-install/reboot_in_progress note
`
