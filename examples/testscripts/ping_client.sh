#!/bin/bash
ping -W 5 -c 1 {{.DestinationPort.Ipv4Address.String}}
ping -W 5 -c 1 {{.DestinationPort.Ipv4Address.String}}
ping -W 5 -c 1 {{.DestinationPort.Ipv4Address.String}}
ping -c 4 {{.DestinationPort.Ipv4Address.String}}
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
