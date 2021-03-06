package main

import (
	"bytes"
	"text/template"
)

func executeTemplate(tmplStr string, data interface{}) string {
	var out bytes.Buffer

	tmpl := template.Must(template.New("script").Parse(tmplStr))
	err := tmpl.Execute(&out, data)
	if err != nil {
		panic(err)
	}

	return out.String()
}
