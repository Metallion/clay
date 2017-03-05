#!/bin/bash
nc -v -z -w 1 {{.DestinationPort.Ipv4Address.String}} 22
{{if eq .Accessibility true}}
if [ $? -eq 0 ]; then
  exit 0
else
  exit 1
fi
{{else}}
if [ $? -eq 0 ]; then
  exit 1
else
  exit 0
fi
{{end}}
