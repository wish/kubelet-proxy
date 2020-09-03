package integrationtests

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func runCommand(t *testing.T, name string, args ...string) (success bool, stdout string, stderr string) {
	cmd := exec.Command(name, args...)
	stdoutReader, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stderrReader, err := cmd.StderrPipe()
	require.NoError(t, err)

	success = true
	require.NoError(t, cmd.Start())

	stdoutBytes, err := ioutil.ReadAll(stdoutReader)
	require.NoError(t, err)
	stderrBytes, err := ioutil.ReadAll(stderrReader)
	require.NoError(t, err)

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			success = false
		} else {
			t.Fatal(err)
		}
	}

	return success, string(stdoutBytes), string(stderrBytes)
}

func randomHexString(t *testing.T, length int) string {
	b := length / 2
	randBytes := make([]byte, b)

	if n, err := rand.Reader.Read(randBytes); err != nil || n != b {
		if err == nil {
			err = fmt.Errorf("only got %v random bytes, expected %v", n, b)
		}
		t.Fatal(err)
	}

	return hex.EncodeToString(randBytes)
}
