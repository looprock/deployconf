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
	Hostname string `yaml:"hostname,omitempty"`
	Containers []struct {
		Name string `yaml:"name"`
		Image string `yaml:"image,omitempty"`
		Env []struct {
			Name string `yaml:"name"`
			Value string `yaml:"value"`
		} `yaml:"env,omitempty"`
		Portnumber int `yaml:"portnumber"`
		Protocol string `yaml:"protocol"`
		Probes []struct {
			Tcpready bool `yaml:"tcpready,omitempty"`
			Tcplive bool `yaml:"tcplive,omitempty"`
			Httpcheck bool `yaml:"httpcheck,omitempty"`
		} `yaml:"probes,omitempty"`
	} `yaml:"containers"`
}

var deploy = `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: {{.Name}}
  name: {{.Name}}
spec:
  replicas: 2
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
{{if .Probes}}
{{range .Probes}}
{{if .Tcplive}}
        livenessProbe:
          failureThreshold: 3
          initialDelaySeconds: 30
          periodSeconds: 30
          successThreshold: 1
          tcpSocket:
            port: {{$portnumber}}
          timeoutSeconds: 10
{{end}}
{{if .Tcpready}}
        readinessProbe:
          failureThreshold: 3
          initialDelaySeconds: 30
          periodSeconds: 10
          successThreshold: 1
          tcpSocket:
            port: {{$portnumber}}
          timeoutSeconds: 10
{{end}}
{{if .Httpcheck}}
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
        name: {{.Name}}
        ports:
        - containerPort: {{$portnumber}}
          name: {{.Name}}-{{$portnumber}}
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
  ports:
{{if eq "true" .Servicetarget}}
{{range .Containers}}
{{$portnumber := .Portnumber}}
{{$protocol := .Protocol}}
  - name: {{.Name}}
    port: {{$portnumber}}
    protocol: {{$protocol}}
    targetPort: {{$portnumber}}
{{end}}
{{else}}
{{$servicetarget := .Servicetarget}}
{{range .Containers}}
{{$portnumber := .Portnumber}}
{{$protocol := .Protocol}}
{{if eq $servicetarget .Name}}
    - name: {{.Name}}
      port: {{$portnumber}}
      protocol: {{$protocol}}
      targetPort: {{$portnumber}}
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

    // if hostname is set, process ingress
    // execute curl to create dns entry
    // htf modify the ingress entry?
    // unmarshal, check if exists, if not, add, and re-marshal?

    // inject all this into kubernetes API instead of using files
}
