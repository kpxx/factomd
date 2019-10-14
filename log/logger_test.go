package log_test

import (
	"github.com/FactomProject/factomd/log"
	"github.com/FactomProject/factomd/registry"
	"github.com/FactomProject/factomd/worker"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLogPrintf(t *testing.T) {
	assert.NotPanics(t, func() {
		log.LogPrintf("testing", "unittest %v", "FOO")
	})
}
func TestRegisterThread(t *testing.T) {

	threadFactory := func(w *worker.Thread, args ...interface{}) {
		assert.NotPanics(t, func() {
			w.Log.LogPrintf("testing", "%v", "foo")
		})
	}
	// create a process with 3 root nodes
	p := registry.New()
	p.Register(threadFactory)
	p.Run()
}