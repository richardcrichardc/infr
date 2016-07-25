# This script is supposed to be idempotent

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

cat <<'EOF' > /etc/ssh/ssh_known_hosts
{{.KnownHosts}}
EOF

# Configure HAProxy

cat <<'EOF' > /etc/haproxy/errors/no-backend.http
HTTP/1.0 404 Service Unavailable
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>404 Not Found</h1>
No such site.
</body></html>

EOF

cat <<'EOF' > /etc/haproxy/haproxy.cfg
{{.host.HAProxyCfg}}
EOF

cat <<'EOF' > /etc/haproxy/https-domains
{{.host.HAProxyHttpsDomains}}
EOF

/etc/haproxy/issue-ssl-certs richard@tawherotech.nz
/etc/haproxy/install-ssl-certs

service haproxy reload
