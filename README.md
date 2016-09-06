#INFR

## Introduction

To be done

## Architecture Overview

To be done

## Getting Started

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

###Adding your first Host  

Now that you have your minimal configuration, you can add a host. 

First you will need a server to use as a Host. Infr is designed to use VPSs from VPS providers such as Linode, Digital Ocean and Vultr, it can however use any Linux servers on the public internet.

So go to your favourite VPS provider and spin up a host. It does not matter what distribution you select as we will be replacing it immediately to get a BTRFS filesystem used by the backup. All you need is to know the host's IP address abd have root access via SSH, using the public key you added above, or with a password. DO NOT USE A MACHINE WITH DATA ON IT YOU WANT TO KEEP, THE ENTIRE HARD DISK WILL BE ERASED. 

Once the machine is up you can add it to the machines managed by infr by running the command:

    $ infr hosts add [-p <root-password>] <name> <ip-address>

Where:

 * `root-password` is the the machine's root password, this can be skipped if your SSH key is authorised to login as root
 *  `name` is what you want to call the host, and 
 *  `ip-address` is the machines public ip address
 
This will start the bootstrap process which replaces the hosts operating system with Debian 8 on a BTRFS filesystem. It takes 10-15 minutes, look [here](how-hosts-are-installed) if you want to know how it works.

Once your host is set up you can list your hosts:

    $ infr hosts
    NAME            PUBLIC IP       PRIVATE IP
    ==========================================
    bob             45.32.191.151   10.8.1.1   
    
And view the DNS records:
    
    $ infr dns
    FQDN                                     TYPE  VALUE           TTL     FOR                 STATUS
    =================================================================================================
    bob.infr.tawherotech.nz               A     45.32.191.151   3600    HOST PUBLIC IP      ???   
    bob.vnet.tawherotech.nz               A     10.8.1.1        3600    HOST PRIVATE IP     ???   

    DNS records are not automatically managed, set 'dnsProvider' and related config settings to enable.

We have not configured a DNS Provider yet, so these records have a status of '???'. 

For now if you want to have a look around the server, SSH in using the IP address, the user 'manager' has been set up with your SSH keys.

### Adding your First LXC

To be done

### Setting up a DNS Provider

To be done

##How it Works

### <a name="how-hosts-are-installed"></a>How Hosts are installed

1. First Infr will download and build iPXE which is used to start a network install of Debian. (This will fail if you don't have a C compiler set up - run `sudo apt-get install build essential` or similar if you have problems.)
2. Next it will upload the configuration settings for the installer to an anonymous Gist on Github. (This includes the SSH public key you specified above - if this worries you review how public key crypto works before running the `hosts add` command.)
3. Then Infr will SCP iPXE up to the host.
4. Next writes to the hosts hard disk will be frozen, iPXE will be written over the bootblock of the hard disk and the host will be rebooted.
5. When the host reboots, iPXE will start a network install of Debian with the configuration uploaded in the anonymous GIST. The machine will _go dark_ during the install - you can monitor it using the VPSs virtual console. Critically, the network install:
    * Sets up BTRFS as the filesystem, which is used for creating backups
    * Installs your SSH public key so the machine can be accesses remotely
6. Once Debian is installed Infr SSHs in and sets up the host, essentially installing HAProxy and setting it's initial configuration.

Once 


  
  


