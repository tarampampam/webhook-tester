package serve

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddress_IsValidValue(t *testing.T) {
	tests := []struct {
		giveAddr  string
		wantError bool
	}{
		{giveAddr: "1.1.1.1", wantError: false},
		{giveAddr: "foobar", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.giveAddr, func(t *testing.T) {
			err := new(address).IsValidValue(tt.giveAddr)

			if tt.wantError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestPublicDir_IsValidValue(t *testing.T) {
	tmpDir := createTempDir(t)
	defer removeTempDir(t, tmpDir)

	tests := []struct {
		giveDirPath string
		wantError   bool
	}{
		{giveDirPath: tmpDir, wantError: false},
		{giveDirPath: "foobar", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.giveDirPath, func(t *testing.T) {
			err := new(publicDir).IsValidValue(tt.giveDirPath)

			if tt.wantError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func Test_address_String(t *testing.T)   { assert.Equal(t, "0.0.0.0", address("0.0.0.0").String()) }
func Test_port_String(t *testing.T)      { assert.Equal(t, "123", port(123).String()) }
func Test_publicDir_String(t *testing.T) { assert.Equal(t, "foo", publicDir("foo").String()) }

func TestCommand_Execute(t *testing.T) {
	t.Skip("Not implemented yet")
}

// Create temporary directory.
func createTempDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := ioutil.TempDir("", "test-")
	if err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

// Remove temporary directory.
func removeTempDir(t *testing.T, dirPath string) {
	t.Helper()

	if !strings.HasPrefix(dirPath, os.TempDir()) {
		t.Fatalf("Wrong tmp dir path: %s", dirPath)
		return
	}

	if err := os.RemoveAll(dirPath); err != nil {
		t.Fatal(err)
	}
}
