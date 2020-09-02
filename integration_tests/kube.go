package integrationtests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/require"
	"gotest.tools/poll"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	testNamespacePrefix        = "kubelet-proxy-test-"
	k8sResourceNameLengthLimit = 63
)

func kubeClient(t *testing.T) kubernetes.Interface {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig(t))
	require.NoError(t, err)

	client, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	return client
}

// waitForPodToComeUp waits for a pod matching `selector` to come up in `namespace`, and returns it.
func waitForPodToComeUp(t *testing.T, namespace, selector string, pollOps ...poll.SettingOp) (pod *corev1.Pod) {
	client := kubeClient(t)
	listOptions := metav1.ListOptions{LabelSelector: selector}

	pollingFunc := func(_ poll.LogT) poll.Result {
		podList, err := client.CoreV1().Pods(namespace).List(context.Background(), listOptions)
		if err != nil {
			return poll.Error(err)
		}

		switch len(podList.Items) {
		case 0:
			return poll.Continue("no pod matching %s in namespace %s", selector, namespace)
		case 1:
			pod = &podList.Items[0]
			return poll.Success()
		default:
			err = fmt.Errorf("expected no more than 1 pod matching %s in namespace %s, found %d",
				selector, namespace, len(podList.Items))
			return poll.Error(err)
		}
	}

	poll.WaitOn(t, pollingFunc, pollOps...)

	return
}

// createNamespace creates a new namespace, and fails the test if it already exists.
// returns the namespace's name.
func createNamespace(t *testing.T, name string) string {
	fullName := testNamespacePrefix
	if name != "" {
		fullName += name + "-"
	}
	fullName += randomHexString(t, k8sResourceNameLengthLimit-len(fullName))

	runKubectlCommandOrFail(t, "create", "namespace", fullName)

	return fullName
}

func deleteNamespace(t *testing.T, name string) {
	runKubectlCommandOrFail(t, "delete", "namespace", name)
}

func applyManifest(t *testing.T, path string) {
	runKubectlCommandOrFail(t, "apply", "-f", path)
}

func deleteManifest(t *testing.T, path string) {
	runKubectlCommandOrFail(t, "delete", "-f", path)
}

func runKubectlCommand(t *testing.T, args ...string) (success bool, stdout string, stderr string) {
	kubeconfigArg := fmt.Sprintf("--kubeconfig=%s", kubeconfig(t))
	return runCommand(t, kubectl(t), append([]string{kubeconfigArg}, args...)...)
}

func runKubectlCommandOrFail(t *testing.T, args ...string) {
	success, stdout, stderr := runKubectlCommand(t, args...)
	if !success {
		t.Fatal(stdout, stderr)
	}
	fmt.Print(stdout)
}

func kubectl(t *testing.T) string {
	return fromEnv(t, "KUBECTL", func(t *testing.T) string {
		return "kubectl"
	})
}

func kubeconfig(t *testing.T) string {
	return fromEnv(t, "KUBECONFIG", func(t *testing.T) string {
		path, err := homedir.Expand("~/.kube/config")
		require.NoError(t, err)
		return path
	})
}

func fromEnv(t *testing.T, key string, defaultValueFactory func(*testing.T) string) string {
	if fromEnv := os.Getenv(key); fromEnv != "" {
		return fromEnv
	}
	return defaultValueFactory(t)
}
