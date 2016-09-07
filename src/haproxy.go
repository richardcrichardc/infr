package main

import (
	"strings"
)

func (h *host) HAProxyCfg() string {
	return executeTemplate(haproxyCfgTmpl, &haproxyCfgData{host: h})
}

func (h *host) HAProxyHttpsDomains() string {
	var fqdns []string

	for _, lxc := range h.AllLxcs() {
		if lxc.Https == HTTPSTERMINATE {
			fqdns = append(fqdns, lxc.FQDN())
			fqdns = append(fqdns, lxc.Aliases...)
		}
	}

	return strings.Join(fqdns, "\n")
}

type haproxyCfgData struct {
	*host
}

const haproxyCfgTmpl = `
global
        log /dev/log    local0
        log /dev/log    local1 notice
        chroot /var/lib/haproxy
        stats socket /run/haproxy/admin.sock mode 660 level admin
        stats timeout 30s
        user haproxy
        group haproxy
        daemon

        # Default SSL material locations
        ca-base /etc/ssl/certs
        crt-base /etc/ssl/private

        # Default ciphers to use on SSL-enabled listening sockets.
        # For more information, see ciphers(1SSL). This list is from:
        #  https://hynek.me/articles/hardening-your-web-servers-ssl-ciphers/
        ssl-default-bind-ciphers ECDH+AESGCM:DH+AESGCM:ECDH+AES256:DH+AES256:ECDH+AES128:DH+AES:ECDH+3DES:DH+3DES:RSA+AESGCM:RSA+AES:RSA+3DES:!aNULL:!MD5:!DSS
        ssl-default-bind-options no-sslv3

defaults
        log     global
        mode    http
        option  httplog
        option  dontlognull
        timeout connect 5000
        timeout client  50000
        timeout server  50000
        errorfile 400 /etc/haproxy/errors/400.http
        errorfile 403 /etc/haproxy/errors/403.http
        errorfile 408 /etc/haproxy/errors/408.http
        errorfile 500 /etc/haproxy/errors/500.http
        errorfile 502 /etc/haproxy/errors/502.http
        errorfile 503 /etc/haproxy/errors/503.http
        errorfile 504 /etc/haproxy/errors/504.http


frontend http
        bind {{.host.PublicIPv4}}:80
        mode http
        option httplog

        use_backend certbot if { path_beg /.well-known/ }

{{ range .host.AllLxcs -}}
    {{- if .HttpBackend }}
        use_backend {{.HttpBackend}} if { hdr(Host) -i {{.FQDN}} {{ range .Aliases -}}{{ . }} {{ end }} }
    {{- end -}}
{{- end }}

        default_backend no_backend


frontend https
        bind {{.host.PublicIPv4}}:443 ssl crt /etc/haproxy/ssl/default.crt crt-list /etc/haproxy/ssl-crt-list
        mode http
        option httplog
{{ range .host.AllLxcs -}}
    {{- if .HttpsBackend }}
        use_backend {{.HttpsBackend}} if { hdr(Host) -i {{.FQDN}} {{ range .Aliases -}}{{ . }} {{ end }} }
    {{- end -}}
{{- end }}

{{ range .host.AllLxcs }}
backend {{.Name}}_http
        server {{.Name}} {{.PrivateIPv4}}:{{.HttpPort}}
{{ end }}

{{ $host := .host -}}
{{ range .host.AllLxcs -}}
    {{- $lxc := . -}}
    {{- range .TCPForwards }}
listen forward_{{$host.Name}}:{{.HostPort}}_{{$lxc.Name}}:{{.HostPort}}
        mode tcp
        bind 0.0.0.0:{{.HostPort}}
        server {{$lxc.Name}}:{{.LxcPort}} {{$lxc.PrivateIPv4}}:{{.LxcPort}}
    {{ end -}}
{{- end }}

backend certbot
        server localhost 127.0.0.1:9980

backend redirect_https
        redirect scheme https

backend no_backend
        errorfile 503 /etc/haproxy/errors/no-backend.http
`
