package main

import ()

func (h *host) HAProxyCfg() string {
	return executeTemplate(haproxyCfgTmpl, &haproxyCfgData{host: h})
}

func (h *host) HttpsTerminateDomains() []string {
	var fqdns []string

	for _, lxc := range h.AllLxcs() {
		_ = lxc
		if lxc.Https == HTTPSTERMINATE {
			fqdns = append(fqdns, lxc.FQDN())
			fqdns = append(fqdns, lxc.Aliases...)
		}
	}

	return fqdns
}

type haproxyCfgData struct {
	*host
}

const haproxyCfgTmpl = `
global
        log /dev/log    local0
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

        use_backend certbot_backend if { path_beg /.well-known/acme-challenge/ } { hdr(Host) -i {{ range .host.HttpsTerminateDomains -}}{{ . }} {{ end }} }

{{ range .host.AllLxcs -}}
    {{- if .HttpBackend }}
        use_backend {{.HttpBackend}} if { hdr(Host) -i {{.FQDN}} {{ range .Aliases -}}{{ . }} {{ end }} }
    {{- end -}}
{{- end }}

        default_backend no_backend


frontend https
        bind {{.host.PublicIPv4}}:443
        mode tcp
        option tcplog
        option logasap

        tcp-request inspect-delay 2s
        tcp-request content reject if { req.ssl_ver 3 }

{{ range .host.AllLxcs -}}
    {{- if .HttpsBackend }}
        use_backend {{.HttpsBackend}} if { req.ssl_sni -i {{.FQDN}} {{ range .Aliases -}}{{ . }} {{ end }} }    {{- end -}}
{{- end }}


frontend https_terminate
        bind 127.0.0.1:443 ssl crt /etc/haproxy/ssl/default.crt crt-list /etc/haproxy/ssl-crt-list
        mode http
        option httplog
{{ range .host.AllLxcs -}}
    {{- if .HttpsTerminate }}
        use_backend {{.Name}}_http if { hdr(Host) -i {{.FQDN}} {{ range .Aliases -}}{{ . }} {{ end }} }
    {{- end -}}
{{- end }}

        default_backend no_backend


{{ range .host.AllLxcs }}
    {{- if or .HttpForward .HttpsTerminate }}
backend {{.Name}}_http
        server {{.Name}} {{.PrivateIPv4}}:{{.HttpPort}}
    {{- end }}
{{ end }}

{{ range .host.AllLxcs }}
    {{- if .HttpsForward }}
backend {{.Name}}_https
        mode tcp
        server {{.Name}} {{.PrivateIPv4}}:{{.HttpsPort}}
    {{- end }}
{{ end }}

{{ $host := .host -}}
{{ range .host.AllLxcs -}}
    {{- $lxc := . -}}
    {{- range .TCPForwards }}
listen forward_{{$host.Name}}:{{.HostPort}}_{{$lxc.Name}}:{{.HostPort}}
        mode tcp
        option tcplog
        option logasap

        timeout client 3600s
        timeout server 3600s
        bind 0.0.0.0:{{.HostPort}}
        server {{$lxc.Name}}:{{.LxcPort}} {{$lxc.PrivateIPv4}}:{{.LxcPort}}
    {{ end -}}
{{- end }}

backend certbot_backend
        server certbot_server 127.0.0.1:9980

backend redirect_https
        redirect scheme https

backend loop_https_terminate
        mode tcp
        server https_terminate_backend 127.0.0.1:443

backend no_backend
        errorfile 503 /etc/haproxy/errors/no-backend.http
`
