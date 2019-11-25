package leader

import (
	"encoding/binary"
	"fmt"
	"github.com/FactomProject/factomd/common/constants"
	"github.com/FactomProject/factomd/common/interfaces"
	"github.com/FactomProject/factomd/common/messages"
	"github.com/FactomProject/factomd/common/primitives"
	llog "github.com/FactomProject/factomd/log"
	"github.com/FactomProject/factomd/modules/event"
	"github.com/FactomProject/factomd/state"
)

var log = llog.PackageLogger

type Leader struct {
	Pub
	Sub
	*Events          // events indexed by VM
	VMIndex   int    // vm this leader is responsible fore
	SysHeight uint32 // KLUDGE don't need this but here for legacy reasons
	Height    uint32 // height leader should next ack
	Minute    int    // current minute
}

// initialize the leader event aggregate
func New(s *state.State) *Leader {
	// TODO: track Db Height so we can decide whether to send out dbsigs
	l := new(Leader)
	l.VMIndex = s.LeaderVMIndex
	{
		pl := s.ProcessLists.Get(s.LLeaderHeight)
		l.SysHeight = uint32(pl.System.Height)
	}
	l.Minute = s.CurrentMinute

	l.Events = &Events{
		Config: &event.LeaderConfig{
			Salt:            s.Salt,
			IdentityChainID: s.IdentityChainID,
			ServerPrivKey:   s.ServerPrivKey,
			FactomSecond:    s.FactomSecond(),
		},
		DBHT:      nil,
		Balance:   nil,
		Directory: nil,
		EOM:       nil,
		Ack:       nil,
	}

	return l
}

func (l *Leader) SendOut(msg interfaces.IMsg) {
	log.LogMessage("leader.txt", "sendout", msg)
	l.Pub.MsgOut.Write(msg)
}

func (l *Leader) Sign(b []byte) interfaces.IFullSignature {
	return l.Config.ServerPrivKey.Sign(b)
}

// Returns a millisecond timestamp
func (l *Leader) GetTimestamp() interfaces.Timestamp {
	return primitives.NewTimestampNow()
}

func (l *Leader) GetSalt(ts interfaces.Timestamp) uint32 {
	var b [32]byte
	copy(b[:], l.Config.Salt.Bytes())
	binary.BigEndian.PutUint64(b[:], uint64(ts.GetTimeMilli()))
	c := primitives.Sha(b[:])
	return binary.BigEndian.Uint32(c.Bytes())
}

func (l *Leader) LeaderExecute(m interfaces.IMsg) {
	switch m.Type() {
	case constants.DIRECTORY_BLOCK_SIGNATURE_MSG:
	default:
		panic(fmt.Sprintf("Unsupported msg %v", m.Type()))
	}

	ack := l.NewAck(m, l.BalanceHash).(*messages.Ack) // LeaderExecute
	m.SetLeaderChainID(ack.GetLeaderChainID())        //  REVIEW: this seems starnge
	m.SetMinute(ack.Minute)
	l.SendOut(m)
}
