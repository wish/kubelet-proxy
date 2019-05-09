package main

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

var opts struct {
	LogLevel           string   `long:"log-level" env:"LOG_LEVEL" description:"Log level" default:"info"`
	BindAddr           string   `long:"bind-address" env:"BIND_ADDRESS" description:"address for binding proxy to" default:":10255"`
	APIEndpoint        string   `long:"kubelet-api" env:"KUBELET_API" description:"kubelet API endpoint" default:"https://localhost:10250"`
	InsecureSkipVerify bool     `long:"kubelet-api-insecure-skip-verify" env:"KUBELET_API_INSECURE_SKIP_VERIFY" description:"skip verification of TLS certificate from kubelet API"`
	Paths              []string `long:"paths" env:"KUBELET_PROXY_PATHS" description:"paths to allow proxying"`
	Methods            []string `long:"methods" env:"KUBELET_PROXY_METHODS" description:"methods to allow proxying" default:"GET"`
}

func getClient() (*http.Client, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		if err == rest.ErrNotInCluster {
			if opts.InsecureSkipVerify {
				tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
				return &http.Client{Transport: tr}, nil
			}

			return http.DefaultClient, nil
		}
		return nil, err
	}
	if opts.InsecureSkipVerify {
		config.TLSClientConfig.Insecure = true
		config.TLSClientConfig.CAData = nil
		config.TLSClientConfig.CAFile = ""
	}
	transport, err := rest.TransportFor(config)
	if err != nil {
		return nil, err
	}

	return &http.Client{Transport: transport}, nil
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		// If the error was from the parser, then we can simply return
		// as Parse() prints the error already
		if _, ok := err.(*flags.Error); ok {
			os.Exit(1)
		}
		logrus.Fatalf("Error parsing flags: %v", err)
	}

	// Use log level
	level, err := logrus.ParseLevel(opts.LogLevel)
	if err != nil {
		logrus.Fatalf("Unknown log level %s: %v", opts.LogLevel, err)
	}
	logrus.SetLevel(level)

	// Set the log format to have a reasonable timestamp
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	logrus.SetFormatter(formatter)

	// Create a map for quick lookups
	methods := make(map[string]struct{})
	for _, m := range opts.Methods {
		methods[m] = struct{}{}
	}

	paths := make(map[string]struct{})
	for _, p := range opts.Paths {
		paths[p] = struct{}{}
	}
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If the method isn't allowed
			if _, ok := methods[r.Method]; !ok {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}

			// If the path is not in our list, return 404
			if _, ok := paths[r.URL.Path]; !ok {
				http.NotFound(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// get HTTP client from k8s
	client, err := getClient()
	if err != nil {
		logrus.Fatal(err)
	}

	rpURL, err := url.Parse(opts.APIEndpoint)
	if err != nil {
		logrus.Fatal(err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(rpURL)
	reverseProxy.Transport = client.Transport
	origDirector := reverseProxy.Director
	reverseProxy.Director = func(r *http.Request) {
		origDirector(r)
		// Override the scheme to match the proxy destination
		r.URL.Scheme = rpURL.Scheme
	}

	logrus.Infof("Starting proxy on %s", opts.BindAddr)

	logrus.Fatal(http.ListenAndServe(opts.BindAddr, middleware(reverseProxy)))
}
