package integrationtests

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/poll"
	corev1 "k8s.io/api/core/v1"
)

const (
	tmpRoot = "tmp"
)

// tests that proxying /metrics works
func TestHappyPathWithMetrics(t *testing.T) {
	namespace, tearDown := integrationTestSetup(t, "happy-path-metrics")
	defer tearDown()

	pod := waitForPodToComeUp(t, namespace, "app=kube-api-proxy")

	stdout := runCurlCommand(t, namespace, pod, "localhost:10255/metrics")

	// there should be _a lot_ of metrics
	lines := strings.Split(stdout, "\n")
	if assert.True(t, len(lines) > 100) {
		// and one of them should be "apiserver_request_duration_seconds"
		for _, line := range lines {
			if strings.HasPrefix(line, "apiserver_request_duration_seconds") {
				return
			}
		}
		t.Errorf("Did not find metric apiserver_request_duration_seconds")
	}
}

func TestForbidden(t *testing.T) {
	namespace, tearDown := integrationTestSetup(t, "forbidden")
	defer tearDown()

	pod := waitForPodToComeUp(t, namespace, "app=kube-api-proxy")

	t.Run("with a forbidden path", func(t *testing.T) {
		stdout := runCurlCommand(t, namespace, pod, "-o /dev/null -s localhost:10255/not_metrics -w '%{http_code}'")
		assert.Equal(t, "404", stdout)
	})

	t.Run("with a forbidden method", func(t *testing.T) {
		stdout := runCurlCommand(t, namespace, pod, "-o /dev/null -s -XPOST localhost:10255/metrics -w '%{http_code}'")
		assert.Equal(t, "405", stdout)
	})
}

// integrationTestSetup creates a new namespace to play in, and returns the name of that namespace
// together with a function to tear it down afterwards.
// It also deploys the proxy in that namespace, with the given flags
func integrationTestSetup(t *testing.T, name string) (string, func()) {
	if _, err := os.Stat(tmpRoot); os.IsNotExist(err) {
		require.NoError(t, os.Mkdir(tmpRoot, os.ModePerm))
	}
	tmpDir, err := ioutil.TempDir(tmpRoot, name+"-")
	require.NoError(t, err)

	namespace := createNamespace(t, name)

	manifestPath := renderProxyDaemonSetManifest(t, tmpDir, namespace)
	applyManifest(t, manifestPath)

	tearDownFunc := func() {
		deleteManifest(t, manifestPath)
		deleteNamespace(t, namespace)
		require.NoError(t, os.RemoveAll(tmpDir))
	}

	return namespace, tearDownFunc
}

func renderProxyDaemonSetManifest(t *testing.T, tmpDir, namespace string) string {
	tplName := "manifest.yml"

	tpl, err := template.New(tplName).Parse(manifestTemplate)
	require.NoError(t, err)

	renderedTemplate, err := os.Create(path.Join(tmpDir, tplName))
	require.NoError(t, err)
	defer renderedTemplate.Close()

	vars := map[string]string{
		"Namespace":          namespace,
		"ServiceAccountName": "kubelet-proxy-sa",
	}

	require.NoError(t, tpl.Execute(renderedTemplate, vars))

	return renderedTemplate.Name()
}

func runCurlCommand(t *testing.T, namespace string, pod *corev1.Pod, curlCommand string) (stdout string) {
	poll.WaitOn(t, func(_ poll.LogT) poll.Result {
		ok, out, stderr := runKubectlCommand(t,
			fmt.Sprintf("--namespace=%s", namespace),
			"exec",
			pod.Name,
			"--",
			"sh", "-c",
			"apk add curl > /dev/null && curl "+curlCommand)

		if ok {
			stdout = strings.TrimSpace(out)
			return poll.Success()
		}

		return poll.Continue("command failed; stdout: %q and stderr %q", stdout, stderr)
	})

	return
}
