#!/bin/bash
{{- $destinationMap := map}}
{{- if eq .DestinationPort.ID 0}}
{{- $destinationMap := putToMap $destinationMap "ipv4Address" "8.8.8.8"}}
{{- else}}
{{- $destinationMap := putToMap $destinationMap "ipv4Address" .DestinationPort.Ipv4Address.String}}
{{- end}}
ping -W 5 -c 1 {{$destinationMap.ipv4Address}}
ping -W 5 -c 1 {{$destinationMap.ipv4Address}}
ping -W 5 -c 1 {{$destinationMap.ipv4Address}}
ping -c 4 {{$destinationMap.ipv4Address}}
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
