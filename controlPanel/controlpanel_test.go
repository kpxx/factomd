package controlpanel

import (
	"fmt"
	"github.com/FactomProject/factomd/fnode"
	"github.com/FactomProject/factomd/pubsub"
	"github.com/FactomProject/factomd/testHelper"
	"testing"
	"time"
)

// test for live testing
func testControlPanelLive(t *testing.T) {
	// register the fnode so it can be retrieved
	s := testHelper.CreateEmptyTestState()
	fnode.New(s)

	// register the publisher to start the control panel
	_ = pubsub.PubFactory.Threaded(5).Publish(pubsub.GetPath(s.FactomNodeName, "bmv", "rest"))
	p := pubsub.PubFactory.Threaded(5).Publish("test")
	go p.Start()

	go func() {
		i := 1
		for {
			p.Write(fmt.Sprintf("data: %d", i))
			time.Sleep(2 * time.Second)
			i++
		}
	}()

	config := &Config{
		NodeName: s.FactomNodeName,
		Version:  s.FactomdVersion,
		Port:     3000,
	}
	New(config)

	select {}
}
