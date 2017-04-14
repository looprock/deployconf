Create a bootstrap deployment and service for kubernetes based on a minimum viable config

# Usage
deployconf -config=./your.yaml -environment=[beta/prod/etc]

This will create a bootstrap config directory under ./k8s/[environment].

# Configuration Examples

## Minimum Viable Configuration
```
name: foo
containers:
- name: ui
  portnumber: 80
  protocol: TCP
```

## Minimum Viable Configuration with a service
```
name: foo
servicetarget: true
containers:
- name: ui
  portnumber: 80
  protocol: TCP
```
## single container configuration

see file: single-container-example.yaml

## Advanced Configuration

see file: multi-container-example.yaml
