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

## Advanced Configuration
```
# name - MANDATORY: Unique application name
name: foo
# replicas - OPTIONAL: number of replicas, defaults to 2
replicas: 3
# servicetarget - OPTIONAL: container to create a service for
# use the container name for a multi-container deployment
# use 'true' for a single container deployment
servicetarget: ui
# containers - MANDATORY: a list of containers to include in each deployment
containers:
# name- MANDATORY: unique name of container type
- name: api
  # image - OPTIONAL: will default to alpine if not present
  # image: docker.ojointernal.com/consumer/app:api-latest
  # env - OPTIONAL: no defaults
  env:
    - name: AUTH_ROOT
      value: https://beta-consumer.ojointernal.com
    - name: FOOBAR
      value: bass
  # portnumber - MANDATORY: port your service listens on
  portnumber: 4000
  # porttype - MANDATORY: protocol of connection (TCP, UDP)
  protocol: TCP
- name: ui
  portnumber: 80
  protocol: TCP
  # probes - OPTIONAL: enable port probe livenessProbe/httpGet
  probes:
    - httpcheck: true
  ```
