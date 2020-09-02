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
	APIEndpoint        *string  `long:"kubelet-api" env:"KUBELET_API" description:"kubelet API endpoint; defaults to the host's kubelet if running inside a k8s pod, or else to https://localhost:10250'"`
	InsecureSkipVerify bool     `long:"kubelet-api-insecure-skip-verify" env:"KUBELET_API_INSECURE_SKIP_VERIFY" description:"skip verification of TLS certificate from kubelet API"`
	Paths              []string `long:"paths" env:"KUBELET_PROXY_PATHS" description:"paths to allow proxying"`
	Methods            []string `long:"methods" env:"KUBELET_PROXY_METHODS" description:"methods to allow proxying" default:"GET"`
}

func main() {
	parseArgs()
	initLogging()

	restConfig, err := getRestConfig()
	if err != nil {
		logrus.Fatalf("Error getting k8s rest config: %v", err)
	}

	httpClient := buildHttpClient(restConfig)
	kubeletBaseURL := getKubeletBaseURL(restConfig)
	reverseProxyURL, err := url.Parse(kubeletBaseURL)
	if err != nil {
		logrus.Fatalf("Unable to parse %q as URL: %v", kubeletBaseURL, err)
	}

	methods := stringSliceToSet(opts.Methods)
	paths := stringSliceToSet(opts.Paths)

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If the method isn't allowed
			if !methods[r.Method] {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}

			// If the path is not in our list, return 404
			if !paths[r.URL.Path] {
				http.NotFound(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(reverseProxyURL)
	reverseProxy.Transport = httpClient.Transport
	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(r *http.Request) {
		originalDirector(r)
		// Override the scheme to match the proxy destination
		r.URL.Scheme = reverseProxyURL.Scheme
	}

	logrus.Infof("Starting proxy to %s listening on %s", kubeletBaseURL, opts.BindAddr)
	logrus.Fatal(http.ListenAndServe(opts.BindAddr, middleware(reverseProxy)))
}

func parseArgs() {
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		// If the error was from the parser, then we can simply return
		// as Parse() prints the error already
		if _, ok := err.(*flags.Error); ok {
			os.Exit(1)
		}
		logrus.Fatalf("Error parsing flags: %v", err)
	}
}

func initLogging() {
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
}

// if error is nil, then config is nil iff we're not running in a k8s cluster
func getRestConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		if err == rest.ErrNotInCluster {
			return nil, nil
		}
		return nil, err
	}

	if opts.InsecureSkipVerify {
		config.TLSClientConfig.Insecure = true
		config.TLSClientConfig.CAData = nil
		config.TLSClientConfig.CAFile = ""
	}

	return config, nil
}

func buildHttpClient(config *rest.Config) *http.Client {
	if config == nil {
		if opts.InsecureSkipVerify {
			transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
			return &http.Client{Transport: transport}
		}

		return http.DefaultClient
	}

	transport, err := rest.TransportFor(config)
	if err != nil {
		logrus.Fatalf("Unable to get transport from config: %v", err)
	}
	return &http.Client{Transport: transport}
}

func getKubeletBaseURL(config *rest.Config) string {
	if opts.APIEndpoint != nil {
		return *opts.APIEndpoint
	}
	if config == nil {
		return "https://localhost:10250"
	}
	return config.Host
}

func stringSliceToSet(s []string) map[string]bool {
	result := make(map[string]bool)
	for _, i := range s {
		result[i] = true
	}
	return result
}
