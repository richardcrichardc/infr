package main

import (
	"bytes"
	"text/template"
)

func (h *host) HAProxyCfg() string {
	data := &haproxyCfgData{
		host: h,
	}

	var out bytes.Buffer

	tmpl := template.Must(template.New("script").Parse(haproxyCfgTmpl))
	err := tmpl.Execute(&out, data)
	if err != nil {
		errorExit("Error executing haproxy template: %s", err)
	}

	return out.String()
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
{{ range .host.AllLxcs -}}
	{{- if .HttpBackend }}
        use_backend {{.HttpBackend}} if { hdr(Host) -i {{.FQDN}} {{ range .Aliases -}}{{ . }} {{ end }} }
	{{- end -}}
{{- end }}

		default_backend no_backend


{{ range .host.AllLxcs -}}
backend {{.Name}}_http
        server {{.Name}} {{.PrivateIPv4}}:80

{{ end }}

backend redirect_https
        redirect scheme https

backend no_backend
        errorfile 503 /etc/haproxy/errors/no-backend.http
`
