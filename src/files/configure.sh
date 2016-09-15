# Exit on error
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

# reload haproxy before getting new certificates so .well-knwon/acme-challenge is enabled for the new domain
service haproxy reload
issue-ssl-certs {{.AdminEmail}}
install-ssl-certs

# reload haproxy after getting new certificates so certificate is used
service haproxy reload
