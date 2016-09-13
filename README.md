# INFR

## Introduction

To be done

## Getting Started

### Installation

#### Binary Download

 * [Latest x86-64 Linux binary](https://tawherotech.nz/infr/infr)

#### Building from source

1. Install [Go](https://golang.org/doc/install)
2. Clone this repo: `git clone https://github.com/richardcrichardc/infr.git`
3. Run the build script: `cd infr; ./build`
4. Run Infr: `./infr`

#### Other Prerequisites

Adding Hosts involves creating a custom build of [iPXE](http://ipxe.org/) so you will need a working C compiler and some libraries on your machine.

If you are running a Debian based distribution you can make sure your have these dependencies by running:

    $ sudo apt-get install build-essential liblzma-dev

### Initial configuration

Before you can use Infr you need a minimal configuration. Run `infr init` and you will be prompted for the essential settings.

Here are the options you will be prompted for:

* `vnetNetwork` (default `10.8.0.0/16`) - Hosts and Lxcs have IP addresses automatically assigned on a virtual private network. This setting determines the range of IP addresses used by the private network.
* `vnetPoolStart` (default `10.8.1.0`) - the first address within `vnetNetwork` that will be allocated.
* `dnsDomain` - domain name used to compose Host and LXC domain names. This is used in conjuction with the next wto settings.
* `dnsPrefix` (default `infr`) - each Host and LXC is allocated a domain name for it's public IP address. E.g. if the LXC is called `bob` dnsPrefix is `infr` and `dnsDomain` is `example.com, the LXC will have a domain name of `bob.infr.example.com` allocated.
* `vnetPrefix` (default `vnet`) - each Host and LXC is allocated a domain name for it's private IP address. E.g. if the LXC is called `bob` vnetPrefix is `vnet` and `dnsDomain` is `example.com, the LXC will have a domain name of `bob.vnet.example.com` allocated.
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
    fozzie          bob             45.32.191.151   10.8.1.2

And find out more about a particular LXC:

    $ ./infr lxc fozzie
    Name:          fozzie
    Host:          bob
    Distro:        ubuntu xenial
    HTTP: 	       NONE
    HTTPS:         NONE
    Aliases:
    LXC Http Port: 80
    TCP Forwards:

Name, Host and Distro are what you specified when setting up the LXC. The rest of the items relate to getting traffic into the LXC fromthe outside world.

### Setting up a DNS Provider

You now have a Host with an LXC on it. Infr will manage the canonical domain names if you let it:

    $ infr dns
    FQDN                                     TYPE  VALUE           TTL     FOR                 STATUS
    =================================================================================================
    bob.infr.example.com                     A     45.32.191.151   3600    HOST PUBLIC IP      ???
    bob.vnet.example.com                     A     10.8.1.1        3600    HOST PRIVATE IP     ???
    fozzie.infr.example.com                  A     45.32.191.151   3600    LXC HOST PUBLIC IP  ???
    fozzie.vnet.example.com                  A     10.8.1.2        3600    LXC PRIVATE IP      ???

    DNS records are not automatically managed, set 'dnsProvider' and related config settings to enable.

These are all the DNS records Infr wants to manage for you. If you configure a DNS Provider, Infr will talk to the DNS Provider's API and configure these for you automatically.

Note they are all subdomains of `infr.example.com` and `vnet.example.com` which you specified when configuring Infr. This is important, as you almost definately have other DNS records in your  `example.com` zone which you don't want Infr to mess with. Infr will not touch any DNS records unless they have one of these two suffixes. However if you manually add DNS records with one of these suffixes on your DNS providers control panel, Infr will happily blast them away.

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

Now that you have a Host, an LXC and working DNS you can SSH into the machines and have a look around. Lets SSH into the Host:

    $ ssh -A manager@bob.infr.example.com

Manager is the shared management account. It has no password so can only be accessed by users who's SSH keys have been added using `ssh keys add <keyfile>`. Manager is a sudoer with the NOPASSWD option, so you can use sudo to administrate the Host.

Note the '-A' option when SSHing into the Host. This enables SSH key forwarding, which means you can now SSH into the the LXCs on the host:

    @bob $ ssh fozzie.vnet.example.com
    
Later on we will set up a virtual network which amoung other things, makes it possible to SSH directly into the LXCs.
    
Note the prompt `@bob $` before the above command. This means you should run this command on the host `bob`. We will use this convention for the rest of the tutorial to show which machine, Host or LXC, that you should run commands on. If the prompt is the plain `$` run the command on the machine you installed Infr.      
    
    
### Exposing services on LXCs to the world

#### Exposing HTTP

Having LXCs and Hosts is all very nice, but it is not very useful until you install some software on a LXC. So lets run a website from fozzie.

@fozzie $ sudo apt-get install apache2

Great, we now have a website. However if we go to http://fozzie.infr.example.com/ in our web browser we will see a `404 Not Found - No such site` error page. This happens because we have not told Infr to forward any traffic from the outside world into `fozzie`. Let's see what forwarding is set up for `bob`.

    $ infr lxc fozzie
    Name:          fozzie
    Host:          bob
    Distro:        ubuntu xenial
    HTTP: 	       NONE
    HTTPS:         NONE
    Aliases:
    LXC Http Port: 80
    TCP Forwards:
    
`HTTP: NONE` indicates HTTP traffic should not be forwarded to the host. You can enable HTTP forwarding by running:

    $ infr lxc fozzie http FORWARD
    
Now browsing to `http://fozzie.infr.example.com/` will show the Apache default page being served from the LXC. HAProxy on `bob` has been configured to forward HTTP traffic for the host names `fozzie.infr.example.com` and `fozzie.vnet.example.com` to port 80 on `fozzie`.

Chances are you want to want your website hosted on a different domain name, let's say `www.example.org`. Outside of infr you will need to set up DNS:

    www.example.org CNAME fozzie.infr.example.org
    
Then tell infr that `www.example.org` is being hosted on `fozzie:

    $ infr lxc fozzie add-alias www.example.org

This will update the HAProxy configuration on `bob` to also forward HTTP traffic for 'www.example.org' to `fozzie`. Aliases can be removed with the `infr lxc <name> remove-alias <alias>` command.

If your HTTP server is running on a different port on the LXC, use the `infr lxc <name> http-port <port>` command to change the port that HTTP traffic will be forwarded to. 

If you want all HTTP traffic redirected to HTTP, use the `infr lxc <name> http <mode>` subcommand to set the HTTP mode to `REDIRECT-HTTPS`.


#### Exposing HTTPS

Note: Some of the examples in this section will not work unless you are have DNS configured - faking it by editing your `/etc/hosts` file will not work.
 
These days you will be wanting to expose all your websites on HTTPS. Infr can deal with HTTPS in two ways:
  
  * Forwarding HTTPS
  * Terminating HTTPS

To enable forwarding of HTTPS traffic use the infr `lxc <name> https <mode>` command to set the https mode to `FORWARD`. HAProxy will be configured to inspect the SNI headers headers and forward traffic for either of the LXC's canonical names or aliases to the LXC's HTTPS port. The HTTPS port defaults to 443, it can be changed with the `lxc <name> https-port command <mode>`.

To enable terminating of HTTPS traffic use the infr `lxc <name> https <mode>` command to set the https mode to `TERMINATE`. The Host will use [Certbot](https://certbot.eff.org/) to aquire a SSL certificate from [Let's Encrypt](https://letsencrypt.org/) and configure HAProxy to terminate SSL, forwarding the decrypted HTTP traffic to the LXC's HTTP port.

Before enabling HTTPS termination you will need to use `lxc config set adminEmail you@example.com` to configure the email address sent to Let's Encrypt. SSL certificates from Let's Encrypt last 3 months - the host will automatically renew the certificates before they expire.

In some circumstances you will want to forward HTTPS traffic to the LXC and have the LXC aquire a SSL certificate from Let's Encrypt. This is not as straightforward as one would hope, some Certbot plugins such as the `apache` plugin tries to authenticate the server over HTTPS - this however leads to a bootstrapping problem, HAProxy cannot determine which LXC to forward HTTPS to until the LXC has a SSL certificate. We have had success using the Certbot standalone plugin with the `--standalone-supported-challenges http-01` option. 

#### Exposing other services

As well as exposing HTTP and HTTPS, infr can expose any service listening on a TCP port of a LXC on it's Host. This is configured with the `infr lxc <host> tcp-forward <host-port> <lxc-port>` command, which configures HAProxy on the Host to do straight TCP proxying.

The main limitation of tcp-forwarding, is that any particular port on a host can only be forwarded to one LXC. To work around this you will need to use non-standard ports if the same service is being exposed on multiple LXC's.
    
### Hooking up the Virtual Network

LXC's are assigned IP addresses on a private network which and canonical domain names are allocated for these IP addresses. The private network is implemented as bridge devices on the Hosts. LXCs on the _same_ Host can communicate to each other using these private network addresses. However in the default configuration, the private network on different Hosts are isolated and cannot be directly accessed from the machines you use for administrating Infr.

Infr can use (ZeroTier One)[https://www.zerotier.com/product-one.shtml] to merge the bridge devices on the Hosts into one combined virtual network. Your adminstration machines can also be attached to the virtual network, they will then be able to access the virtual network from anywhere they have an internet connection even if the internet connection is [broken or heavily restricted](https://www.zerotier.com/config.shtml). 

To 'hook up' the virtual network you will need a Zerotier network controller. Zerotier provide a free hosted service that can be set up in a few minutes:

 1. Sign up at [https://my.zerotier.com/](https://my.zerotier.com/)
 2. Create a new network and configure it, the critical configuration items are:
    * Access Control: Certificate (Private Network)
    * Ethernet Frame Types: 
      * IPv4 (and ARP): On
    * IPv4 Auto-Assign:
      * Auto-Assign from Range: Off
 3. Take note of the Network Id, you will use this to add machines to the virtual network.
      
 To attach the Hosts to the virtual network:
 
 1. Run `infr config set vnetZerotierNetworkId <network-id>`
 2. For each Host run `infr host <name> reconfigure` to tell the host to join the network.
 3. Use the control panel on [https://my.zerotier.com/](https://my.zerotier.com/) to authorise the hosts access to the network and enable it as a bridge. (We may automate this step using the Zerotier API at some point in the future)
 4. If the Host's status on the Zerotier control panel does not chnage to ONLINE try reloading the entire page.

To attach the machines you use for adminitrating Infr to the virtual network:

1. Install [Zerotier One](https://www.zerotier.com/product-one.shtml) on your machine
2. Run `sudo zerotier-cli join <network-id>`
3. Run `sudo zerotier-cli listnetworks` to find out what network interface has been assigned. This will be `zt0` unless you already have Zerotier installed and are attached to other networks
4. Add the snippet of configuration below to `/etc/network/interfaces`, where:
    * `<zerotier-interface>` is the new Zerotier interface - probably `zt0`
    * `<ip-address>` is an unallocated IP address in the virtual network. E.g. if `vnetNetwork` is `10.8.0.0/16` and `vnetPoolStart` is `10.8.1.0` then and IP address between `10.8.0.0` and `10.8.0.255` could be used.
    *  `<network-mask>` is the virtual network's network mask in dot decimal notation, e.g. if vnetNetwork is 10.8.0.0/16 the network mask is /16, here it must be expressed as `255.255.0.0`.
5. Run `sudo ifup <zerotier-interface>`
6. Use the control panel on [https://my.zerotier.com/](https://my.zerotier.com/) to authorise the machine's access to the network.

Snippet to add to `/etc/network/interfaces`:

     auto <zerotier-interface>
     allow-hotplug <zerotier-interface>
     iface <zerotier-interface> inet static
         address <ip-address>
         netmask <network-mask>

Once a Host and your management machine are on the virtual network, you should be able to SSH directly into any LXC on the host:

    $ ssh manager@fozzie.vnet.example.com

To reduce typing make add `dnsDomain` (i.e. `example.com` in these examples) as a search domain to `/etc/resolve.conf` and add add the following to your `~/.ssh/config`:

    Host *.infr
      User manager

    Host *.vnet
      User manager

Now you can SSH into a LXC or Host with:

    $ ssh fozzie.vnet
    
or:

    $ ssh bob.infr 

### Enabling backups

Infr sets up Hosts with BTRFS as the filesystem so we can use btrfs send and btrfs receive to perform [coherant efficient incremental backups](https://btrfs.wiki.kernel.org/index.php/Incremental_Backup) every 5 minutes:
  
  * use of BTRFS snapshots make backups coherant, i.e. as if the state of the entire disc was read at one point in time, ensuring that systems like relational databases that are designed to survive power failure, can reliably be recovered
  * incremental backups are made using `btrfs send` which like backups done with `rsync` use the network efficient, only sending the changes in disc content between the two points in time
  * BTRFS snapshots are 'copy on write' which means the difference between subsequent snapshots can be determined efficiently - tools like `rsync` need to do lot of disc IO to determine what changes have occured
  * the point in time backups are stored as 'copy on write' snapshots, reducing the duplication on disc, making it practical to retain backups at many points in time

The five minute backups are retained as per the following schedule:

 * All backups in the last 2 hours
 * Hourly backups for the last 2 days
 * Daily backups for the last 2 weeks
 * Weekly backups for the last 12 weeks

At least two Hosts are needed before backups can be set up. It is usual to set up a host dedicated to storing backups. We use [Backupsy](https://backupsy.com/) VPSs for our backups.

    $ infr hosts
    NAME            PUBLIC IP       PRIVATE IP
    ==========================================
    bob             45.32.191.151   10.8.1.1
    tom             69.12.95.204    10.8.1.2

Backups are configured using the command `infr backup start <from-host> <to-host>`.

    $ infr backup start bob tom
    $ infr backups
    FROM            TO
    ================================
    bob             tom
    
Currently backup recovery is not automated. Backups can be found on the 'to' Host in the directory `/var/lib/backups/backups/<from-host>`.

### Using Infr with a team    

### Troubleshooting

### Future features

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
