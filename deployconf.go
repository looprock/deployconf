package main

import (
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "fmt"
    "text/template"
    "bytes"
    "regexp"
    "strings"
    "os"
    "flag"
)

type conf struct {
	Name string `yaml:"name"`
	Servicetarget string `yaml:"servicetarget,omitempty"`
  Localservice string `yaml:"localservice,omitempty"`
	Hostname string `yaml:"hostname,omitempty"`
  Replicas string `yaml:"replicas,omitempty"`
	Containers []struct {
		Name string `yaml:"name"`
		Image string `yaml:"image,omitempty"`
    Buildroot string `yaml:"buildroot,omitempty"`
		Env []struct {
			Name string `yaml:"name"`
			Value string `yaml:"value"`
		} `yaml:"env,omitempty"`
		Portnumber int `yaml:"portnumber"`
    Portname string `yaml:"portname,omitempty"`
    Serviceport string `yaml:"serviceport,omitempty"`
		Protocol string `yaml:"protocol"`
		Probes []struct {
			Tcpready bool `yaml:"tcpready,omitempty"`
			Tcplive bool `yaml:"tcplive,omitempty"`
			Httpcheck bool `yaml:"httpcheck,omitempty"`
		} `yaml:"probes,omitempty"`
	} `yaml:"containers"`
}

var gitlabci = `stages:
  - package
  - deploy

{{range .Containers}}
container-{{.Name}}:
  image: docker.company.com/build-images/docker:cli
  stage: package
  services:
    - docker.company.com/build-images/docker:engine
  only:
    - tags
    - master
  variables:
    IMAGE_TAG_BASE: {{.Name}}
  before_script:
    - docker login -u gitlab-ci-token -p $CI_BUILD_TOKEN $CI_REGISTRY
  script:
    {{ if .Buildroot}}
    - export DOCKER_BUILD_ROOT=${DOCKER_BUILD_ROOT:-"{{.Buildroot}}"}
    {{else}}
    - export DOCKER_BUILD_ROOT=${DOCKER_BUILD_ROOT:-"{{.Name}}"}
    {{end}}
    - if [ -n "${CI_BUILD_TAG}" ]; then export DOCKER_CACHE_TAG="latest"; export DOCKER_IMAGE_TAG=${CI_BUILD_TAG}; else export DOCKER_CACHE_TAG="canary"; export DOCKER_IMAGE_TAG="canary"; fi
    - docker pull ${CI_REGISTRY_IMAGE}:${IMAGE_TAG_BASE}-${DOCKER_CACHE_TAG} || echo "Cannot find cache source image. Skipping cache."
    - docker build --pull --cache-from ${CI_REGISTRY_IMAGE}:${IMAGE_TAG_BASE}-${DOCKER_CACHE_TAG} -t ${CI_REGISTRY_IMAGE}:${IMAGE_TAG_BASE}-${DOCKER_IMAGE_TAG} ${DOCKER_BUILD_ROOT}
    - docker push ${CI_REGISTRY_IMAGE}:${IMAGE_TAG_BASE}-${DOCKER_IMAGE_TAG}
    - if [ -n "${CI_BUILD_TAG}" ]; then docker tag ${CI_REGISTRY_IMAGE}:${IMAGE_TAG_BASE}-${DOCKER_IMAGE_TAG} ${CI_REGISTRY_IMAGE}:${IMAGE_TAG_BASE}-latest; docker push ${CI_REGISTRY_IMAGE}:${IMAGE_TAG_BASE}-latest; fi
{{end}}

beta-deploy:
  image: docker.company.com/images/kubectl:latest
  stage: deploy
  environment: beta
  only:
    - master
  script:
    - awk 'FNR==1 && NR!=1 {print "---"}{print}' k8s/beta/* | sed -e "s/##CI_BUILD_NUMBER##/gitlab-build-${CI_PIPELINE_ID}/g" | sed -e "s/##IMAGE_VERSION##/${CI_BUILD_TAG}/g" | kubectl --context=beta --namespace=default apply -l app={{.Name}} -f -
    - kubectl --context=beta --namespace=default rollout status --watch deployment/{{.Name}}

production-deploy:
  image: docker.company.com/images/kubectl:latest
  stage: deploy
  environment: production
  only:
    - /^v\d+\.\d+\.\d+$/
  script:
    - awk 'FNR==1 && NR!=1 {print "---"}{print}' k8s/production/* | sed -e "s/##CI_BUILD_NUMBER##/gitlab-build-${CI_PIPELINE_ID}/g" | sed -e "s/##IMAGE_VERSION##/${CI_BUILD_TAG}/g" | kubectl --context=production --namespace=default apply -l app={{.Name}} -f -
    - kubectl --context=production --namespace=default rollout status --watch deployment/{{.Name}}
`


var deploy = `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: {{.Name}}
  name: {{.Name}}
spec:
{{ if .Replicas}}
  {{if $.Localservice}}
  # overwriting replicas for localservice
  replicas: 1
  {{else}}
  replicas: {{.Replicas}}
  {{end}}
{{else}}
  {{if $.Localservice}}
  # overwriting replicas for localservice
  replicas: 1
  {{else}}
  replicas: 2
  {{end}}
{{end}}
  revisionHistoryLimit: 1
  selector:
    matchLabels:
      app: {{.Name}}
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 50%
    type: RollingUpdate
  template:
    metadata:
      annotations:
        ci.company.com/build-number: ##CI_BUILD_NUMBER##
      creationTimestamp: null
      labels:
        app: {{.Name}}
      name: {{.Name}}
    spec:
      containers:
{{ range .Containers }}
{{if .Env}}
      - env:
{{range .Env}}
        - name: {{.Name}}
          value: {{.Value}}
{{end}}
{{end}}
{{if .Image}}
      {{if .Env}}  image{{else}}- image{{end}}: {{.Image}}
{{else}}
      {{if .Env}}  image{{else}}- image{{end}}: alpine
{{end}}

{{$portnumber := .Portnumber}}
{{$protocol := .Protocol}}
        imagePullPolicy: Always
{{if $portnumber}}
        {{if $.Localservice}}
        # Omitting livenessProbe for localservice
        {{else}}
        livenessProbe:
          failureThreshold: 3
          initialDelaySeconds: 30
          periodSeconds: 30
          successThreshold: 1
          tcpSocket:
            port: {{$portnumber}}
          timeoutSeconds: 10
          {{end}}
{{end}}
{{if $portnumber}}
        {{if $.Localservice}}
        # Omitting readinessProbe for localservice
        {{else}}
        readinessProbe:
          failureThreshold: 3
          initialDelaySeconds: 30
          periodSeconds: 10
          successThreshold: 1
          tcpSocket:
            port: {{$portnumber}}
          timeoutSeconds: 10
          {{end}}
{{end}}
{{if .Probes}}
{{range .Probes}}
{{if .Httpcheck}}
        {{if $.Localservice}}
        # Omitting http livenessProbe probe for localservice
        {{else}}
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /
            port: {{$portnumber}}
            scheme: HTTP
          initialDelaySeconds: 30
          periodSeconds: 30
          successThreshold: 1
          timeoutSeconds: 10
          {{end}}
{{end}}
{{end}}
{{end}}
        name: {{.Name}}
        ports:
        - containerPort: {{$portnumber}}
          {{if .Portname}}
          name: {{.Portname}}
          {{end}}
          protocol: {{$protocol}}
        resources: {}
        terminationMessagePath: /dev/termination-log
{{end}}
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      securityContext: {}
      terminationGracePeriodSeconds: 30`

var service = `apiVersion: v1
kind: Service
metadata:
  {{if eq "true" .Servicetarget}}
  name: {{.Name}}
  labels:
    app: {{.Name}}
  {{else}}
  name: {{.Name}}-{{.Servicetarget}}
  {{end}}
spec:
  selector:
    app: {{.Name}}
  {{if $.Localservice}}
  # adding nodePort config for Localservice
  type: NodePort
  {{end}}
  ports:
{{if eq "true" .Servicetarget}}
{{range .Containers}}
  - name: {{.Name}}
    {{if .Serviceport}}
    port: {{.Serviceport}}
    {{else}}
    port: {{.Portnumber}}
    {{end}}
    protocol: {{.Protocol}}
    targetPort: {{.Portnumber}}
{{end}}
{{else}}
{{$servicetarget := .Servicetarget}}
{{range .Containers}}
{{if eq $servicetarget .Name}}
    - name: {{.Name}}
      {{if .Serviceport}}
      port: {{.Serviceport}}
      {{else}}
      port: {{.Portnumber}}
      {{end}}
      protocol: {{.Protocol}}
      targetPort: {{.Portnumber}}
{{end}}
{{end}}
{{end}}`

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func (c *conf) getConf(conffile string) *conf {
    yamlFile, err := ioutil.ReadFile(conffile)
    check(err)
    err = yaml.Unmarshal(yamlFile, c)
    check(err)
    return c
}

func main() {

    config := flag.String("config", "unset", "YAML configuration")
    environment := flag.String("environment", "unset", "YAML configuration")
    flag.Parse()

    if *config == "unset" {
      fmt.Printf("ERROR: please specify a config file to parse via '-config=' \n")
      os.Exit(1)
    }

    if *environment == "unset" {
      fmt.Printf("ERROR: please specify an environment\n")
      os.Exit(1)
    }

    var c conf
    c.getConf(*config)

    os.Mkdir("k8s", 0755)
    os.Mkdir("k8s/" + *environment, 0755)


    var doc bytes.Buffer
    re := regexp.MustCompile("(?m)^\\s*$[\r\n]*")

    // process deployment
    tmpl, err := template.New("deployment").Parse(deploy)
    check(err)
    tmpl.Execute(&doc, c)
    s := doc.String()
    s = fmt.Sprintf("%v\n", strings.Trim(re.ReplaceAllString(s, ""), "\r\n"))
    b := []byte(s)
    err = ioutil.WriteFile("k8s/" + *environment + "/deployment.yaml", b, 0644)
    check(err)
    fmt.Printf("Created: k8s/" + *environment + "/deployment.yaml\n")

    if c.Servicetarget != "" {
      // process service
      var sdoc bytes.Buffer
      // stmpl, err := template.ParseFiles("test.tmpl")
      stmpl, err := template.New("service").Parse(service)
      check(err)
      stmpl.Execute(&sdoc, c)
      ss := sdoc.String()
      ss = fmt.Sprintf("%v\n", strings.Trim(re.ReplaceAllString(ss, ""), "\r\n"))
      sb := []byte(ss)
      err = ioutil.WriteFile("k8s/" + *environment + "/service.yaml", sb, 0644)
      // fmt.Printf(ss)
      check(err)
      fmt.Printf("Created: k8s/" + *environment + "/service.yaml\n")
    }

    // gitlab-ci config also depends on Servicetarget being set for now
    var gdoc bytes.Buffer
    gtmpl, err := template.New("gitlabci").Parse(gitlabci)
    check(err)
    gtmpl.Execute(&gdoc, c)
    gd := gdoc.String()
    // gd = fmt.Sprintf("%v\n", strings.Trim(re.ReplaceAllString(gd, ""), "\r\n"))
    gb := []byte(gd)
    err = ioutil.WriteFile("example-gitlab-ci.yml", gb, 0644)
    check(err)
    fmt.Printf("Created: example-gitlab-ci.yml\n")
}
