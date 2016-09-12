# INFR

<!-- Authoring note: This file is processed by ronn to create a manpage, ronn does not support H4s and below-->

## Introduction

To be done

## Getting Started

### Binary Download

 * [Latest x86-64 Linux binary](https://tawherotech.nz/infr/infr)


### Building from source

1. Install [Go](https://golang.org/doc/install)
2. Install [Ronn](https://github.com/rtomayko/ronn) (try `apt-get install ruby-ronn`)
3. Clone this repo: `git clone https://github.com/richardcrichardc/infr.git`
4. Run the build script: `cd infr; ./build`
5. Run Infr: `./infr`

### Other Prerequisites

Adding Hosts involves creating a custom build of [iPXE](http://ipxe.org/) so you will need a working C compiler and some libraries on your machine.

If you are running a Debian based distribution you can make sure your have these dependencies by running:

    $ sudo apt-get install build-essential liblzma-dev

### Initial configuration

Before you can use Infr you need a minimal configuration. Run `infr init` and you will be prompted for the essential settings.

Here are the options you will be prompted for:

* `vnetNetwork` (default `10.8.0.0/16`) - Hosts and Lxcs have IP addresses automatically assigned on a virtual private network. This setting determines the range of IP addresses used by the private network.
* `vnetPoolStart` (default `10.8.1.0`) - the first address within `vnetNetwork` that will be allocated.
* `dnsDomain` - domain name used to compose Host and LXC domain names. This is used in conjuction with the next wto settings.
* `dnsPrefix` (default `infr`) - each Host and LXC is allocated a domain name for it's public IP address. E.g. if the LXC is called `bob` dnsPrefix is `infr` and `dnsDomain` is `mydomain.com, the LXC will have a domain name of `bob.infr.mydomain.com` allocated.
* `vnetPrefix` (default `vnet`) - each Host and LXC is allocated a domain name for it's private IP address. E.g. if the LXC is called `bob` vnetPrefix is `vnet` and `dnsDomain` is `mydomain.com, the LXC will have a domain name of `bob.vnet.mydomain.com` allocated.
* Initial SSH public key (default `$HOME/.ssh/id_rsa.pub` - infr maintains a set of SSH keys for everyone who should be able to administrate the Hosts and LXCs.
* `adminEmail` - Will be removing this as a required initial option.

Once set up, these options can be changed using the `infr config` command.

### Adding your first Host

Now that you have your minimal configuration, you can add a host.

First you will need a server to use as the Host. Infr is designed to use VPSs from VPS providers such as Linode, Digital Ocean and Vultr, it can however use any Linux servers on the public internet.

So go to your favourite VPS provider and spin up a host. It does not matter what distribution you select as we will be replacing it immediately to get a BTRFS filesystem used by the backup. All you need is to know the host's IP address abd have root access via SSH, using the public key you added above, or with a password. DO NOT USE A MACHINE WITH DATA ON IT YOU WANT TO KEEP, THE ENTIRE HARD DISK WILL BE ERASED.

Once the machine is up you can add it to the machines managed by infr by running the command:

    $ infr hosts add [-p <root-password>] <name> <ip-address>

Where:

 * `root-password` is the the machine's root password, this can be skipped if your SSH key is authorised to login as root
 *  `name` is what you want to call the host, and
 *  `ip-address` is the machines public ip address

This will start the bootstrap process which replaces the hosts operating system with Debian 8 on a BTRFS filesystem. It takes 10-15 minutes, look [here](#how-hosts-are-installed) if you want to know how hosts are installed.

Once your host is set up you can list your hosts:

    $ infr hosts
    NAME            PUBLIC IP       PRIVATE IP
    ==========================================
    bob             45.32.191.151   10.8.1.1


We have not configured a DNS Provider yet, so these records have a status of '???'. Let's create a LXC first, then we will sort out these DNS records, before SSHing into the Hosts and LXCs.

### Adding your First LXC

Now that you have a Host you can put an LXC on it:

    $ infr lxcs add fozzie ubuntu xenial bob

That will whir away for a minute or two. Then you can list the LXCs:

    $ infr lxcs
    NAME            HOST            PUBLIC IP       PRIVATE IP
    =========================================================
    fozzie          kaiiwi          45.32.191.151   10.8.1.2

And find out more about a particular LXC:

    $ ./infr lxc bob
    Name:          fozzie
    Host:          bob
    Distro:        ubuntu xenial
    Aliases:
    HTTP: 	       NONE
    HTTPS:         NONE
    LXC Http Port: 80
    TCP Forwards:

### Setting up a DNS Provider

You now have a Host with an LXC on it. Infr will manage the canonical domain names if you let it:

    $ infr dns
    FQDN                                     TYPE  VALUE           TTL     FOR                 STATUS
    =================================================================================================
    bob.infr.mydomain.com                    A     45.32.191.151   3600    HOST PUBLIC IP      ???
    bob.vnet.mydomain.com                    A     10.8.1.1        3600    HOST PRIVATE IP     ???
    fozzie.infr.mydomain.com                 A     45.32.191.151   3600    LXC HOST PUBLIC IP  ???
    fozzie.vnet.mydomain.com                 A     10.8.1.2        3600    LXC PRIVATE IP      ???

    DNS records are not automatically managed, set 'dnsProvider' and related config settings to enable.

These are all the DNS records Infr wants to manage for you. If you configure a DNS Provider, Infr will talk to the DNS Provider's API and configure these for you automatically.

Note they are all subdomains of `infr.mydomain.com` and `vnet.mydomain.com` which you specified when configuring Infr. This is important, as you almost definately have other DNS records in your  `mydomain.com` zone which you don't want Infr to mess with. Infr will not touch any DNS records unless they have one of these two suffixes. However if you manually add DNS records with one of these suffixes on your DNS providers control panel, Infr will happily blast them away.

Currently Infr supports two DNS Providers, [Vultr](https://www.vultr.com/) and [Rage4](https://rage4.com/). If you want to use one of these:

1. Log into your DNS Provider, make sure a zone exists for your `dnsDomain` and get your API keys
2. Run `infr config set dnsProvider vultr` or `infr config set dnsProvider rage4` to tell Infr which provider to use
3. Tell Infr what the API keys are:
    * For Vultr you need to use `infr config set` to set `dnsVultrAPIKey`
    * For Rage4 you need to use `infr config set` to set `dnsRage4Account` (to the email address you log into Rage4 with) and `dnsRage4Key`
4. Run `infr dns` to check Infr can access the API. The DNS records should now show up with a Status of MISSING.
5. Run `infr dns fix` to tell Infr to create the missing DNS records

We plan to add new DNS Providers in the future, in particular we are likely to add Linode and Digital Ocean.

If you are unable or don't want to set up a DNS provider right now, you can simulate the result by copying and pasting from `infr DNS` into your `/etc/hosts` file. Doing so will make trying out the rest of Infr more straight forward.

### SSHing into your Hosts and LXCs

Now that you have a Host an LXC and working DNS you can SSH into the machines and have a look around. Lets SSH into the Host:

    $ ssh -A manager@bob.infr.mydomain.com

Manager is the shared management account. It has no password so can only be accessed by users who's SSH keys have been added using `ssh keys add <keyfile>`. Manager is a sudoer with the NOPASSWD option, so you can use sudo to administrate the Host.

Note the '-A' option when SSHing into the Host. This enables SSH key forwarding, which means you can now SSH into the the LXCs on the host:

    $ ssh fozzie.vnet.mydomain.com

## How it Works

### <a name="how-hosts-are-installed"/></a> How Hosts are installed

1. First Infr will download and build iPXE which is used to start a network install of Debian. (This will fail if you don't have a C compiler set up - run `sudo apt-get install build-essential` or similar if you have problems.)
2. Next it will upload the configuration settings for the installer to an anonymous Gist on Github. (This includes the SSH public key you specified above - if this worries you review how public key crypto works before running the `hosts add` command.)
3. Then Infr will SCP iPXE up to the host.
4. Next writes to the hosts hard disk will be frozen, iPXE will be written over the bootblock of the hard disk and the host will be rebooted.
5. When the host reboots, iPXE will start a network install of Debian with the configuration uploaded in the anonymous GIST. The machine will _go dark_ during the install - you can monitor it using the VPSs virtual console. Critically, the network install:
    * Sets up BTRFS as the filesystem, which is used for creating backups
    * Installs your SSH public key so the machine can be accesses remotely
6. Once Debian is installed Infr SSHs in and sets up the host, essentially installing HAProxy and setting it's initial configuration.
