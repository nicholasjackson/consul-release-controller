package testutils

import (
	"io/ioutil"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ClearMockCall(mc *mock.Mock, method string) {
	calls := mc.ExpectedCalls
	new := []*mock.Call{}

	for _, c := range calls {
		if c.Method != method {
			new = append(new, c)
		}
	}

	mc.ExpectedCalls = new
}

func GetTestFilePath(t *testing.T, filename string) string {
	_, b, _, _ := runtime.Caller(0)
	return path.Join(path.Dir(b), "../../test_data", filename)
}

func GetTestData(t *testing.T, filename string) []byte {
	_, b, _, _ := runtime.Caller(0)
	f := path.Join(path.Dir(b), "../../test_data", filename)

	d, err := ioutil.ReadFile(f)
	assert.NoError(t, err)

	return d
}
