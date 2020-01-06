package elections

import (
	"fmt"
	"reflect"
	"time"

	"github.com/FactomProject/factomd/common"
	"github.com/FactomProject/factomd/common/globals"
	"github.com/FactomProject/factomd/common/interfaces"
	"github.com/FactomProject/factomd/common/primitives"
	"github.com/FactomProject/factomd/state"
	"github.com/FactomProject/factomd/util/atomic"
)

var _ = fmt.Print
var _ = time.Tick

var FaultTimeout int = 60 // This value only lasts till the command line is parse which will set it.
var RoundTimeout int = 20 // This value only lasts till the command line is parse which will set it.

type FaultId struct {
	Dbheight int
	Minute   int
	Round    int
}

type Elections struct {
	common.Name
	Manager

	state interfaces.IState
	FedID interfaces.IHash
	//	Name      string
	Sync         []bool                      // List of servers that have Synced
	Federated    []interfaces.IServer        //
	Audit        []interfaces.IServer        //
	FPriority    []interfaces.IHash          //
	APriority    []interfaces.IHash          //
	DBHeight     int                         // Height of this election
	SigType      bool                        // False for dbsig, true for EOM
	Minute       int                         // Minute of this election (-1 for a DBSig)
	VMIndex      int                         // VMIndex of this election
	Msgs         []interfaces.IMsg           // Messages we are collecting in this election.  Look here for what's missing.
	Round        []int                       //
	Electing     int                         // This is the federated Server index that we are looking to replace
	feedback     []string                    //
	VName        string                      //
	Msg          interfaces.IMsg             // The missing message as supplied by the volunteer
	Ack          interfaces.IMsg             // The missing ack for the message as supplied by the volunteer
	Sigs         [][]interfaces.IHash        // Signatures from the Federated Servers for a given round.
	Adapter      interfaces.IElectionAdapter //
	Timeout      time.Duration               // Timeout period before we start the election
	RoundTimeout time.Duration               // Timeout for the next audit to volunteer
	FaultId      atomic.AtomicInt            // Incremented every time we launch a new timeout
}

func New(is interfaces.IState) *Elections {
	e := new(Elections)
	s := is.(*state.State)
	e.NameInit(s, s.GetFactomNodeName()+"Election", reflect.TypeOf(e).String())
	e.state = s
	e.Electing = -1
	e.Timeout = time.Duration(FaultTimeout) * time.Second
	e.RoundTimeout = time.Duration(RoundTimeout) * time.Second
	e.Waiting = make(chan interfaces.IElectionMsg, 500)
	e.Input = s.ElectionsQueue()
	//mgr.Enqueue = func(msg interfaces.IMsg) { mgr.Input.Enqueue(msg) }
	//mgr.Elections.ProcessWaiting = mgr.ProcessWaiting

	// inject reference from state
	s.Elections = e

	return e
}

func (e *Elections) GetFedID() interfaces.IHash {
	return e.FedID
}

func (e *Elections) GetElecting() int {
	return e.Electing
}

func (e *Elections) GetVMIndex() int {
	return e.VMIndex
}

func (e *Elections) GetRound() []int {
	return e.Round
}

func (e *Elections) ComparisonMinute() int {
	if !e.SigType {
		return -1
	}
	return int(e.Minute)
}

func (e *Elections) GetFederatedServers() []interfaces.IServer {
	return e.Federated
}

func (e *Elections) GetAuditServers() []interfaces.IServer {
	return e.Audit
}

func (e *Elections) AddFederatedServer(server interfaces.IServer) int {
	// Already a leader
	if idx := e.GetFedServerIndex(server); idx != -1 {
		return idx
	}

	// If it's an audit server, we need to remove it and add it (promotion)
	e.RemoveAuditServer(server)

	e.Federated = append(e.Federated, server)
	//s := e.State
	//s.LogPrintf("elections", "Election Sort FedServers AddFederatedServer")
	changed := e.Sort(e.Federated)
	if changed {
		e.LogPrintf("election", "Sort changed e.Federated in Elections.AddFederatedServer")
		e.LogPrintLeaders("election")
	}

	return e.GetFedServerIndex(server)
}

func (e *Elections) AddAuditServer(server interfaces.IServer) int {
	// Already an audit server
	if idx := e.GetAudServerIndex(server); idx != -1 {
		return idx
	}

	// If it's a federated server, we need to remove it and add it (promotion)
	e.RemoveFederatedServer(server)

	e.Audit = append(e.Audit, server)
	changed := e.Sort(e.Audit)
	if changed {
		e.LogPrintf("election", "Sort changed e.Audit in Elections.AddAuditServer")
		e.LogPrintLeaders("election")
	}

	return e.GetAudServerIndex(server)
}

func (e *Elections) RemoveFederatedServer(server interfaces.IServer) {
	idx := e.GetFedServerIndex(server)
	if idx == -1 {
		e.RemoveAuditServer(server)
		return
	}

	e.Federated = append(e.Federated[:idx], e.Federated[idx+1:]...)
}

func (e *Elections) RemoveAuditServer(server interfaces.IServer) {
	idx := e.GetFedServerIndex(server)
	if idx == -1 {
		return
	}

	e.Audit = append(e.Audit[:idx], e.Audit[idx+1:]...)
}

func (e *Elections) GetFedServerIndex(server interfaces.IServer) int {
	idx := -1
	for i, s := range e.Federated {
		if s.GetChainID().IsSameAs(server.GetChainID()) {
			idx = i
			break
		}
	}
	return idx
}

func (e *Elections) GetAudServerIndex(server interfaces.IServer) int {
	idx := -1
	for i, s := range e.Audit {
		if s.GetChainID().IsSameAs(server.GetChainID()) {
			idx = i
			break
		}
	}
	return idx
}

func (e *Elections) GetAdapter() interfaces.IElectionAdapter {
	return e.Adapter
}

// Add the given sig list to the list of signatures for the given round.
func (e *Elections) AddSigs(round int, sigs []interfaces.IHash) {
	for len(e.Sigs) <= round {
		e.Sigs = append(e.Sigs)
	}
}

func (e *Elections) NewFeedback() {
	e.feedback = make([]string, len(e.Federated)+len(e.Audit))
}

func (e *Elections) FeedBackStr(v string, fed bool, index int) string {

	if !fed {
		index = index + len(e.Federated)
	}

	// If I have no feedback, then get some.
	if e.feedback == nil || len(e.feedback) == 0 {
		e.NewFeedback()
	}

	// Add the status if it is in my known range.
	if index >= 0 && index < len(e.feedback) {
		e.feedback[index] = v
	}

	// Make a string of the status.
	r := ""
	for _, v := range e.feedback {
		r = r + fmt.Sprintf("%4s ", v)
	}
	if e.Msg != nil {
		r = r + " " + e.VName
		r = r + " " + e.Msg.String()
	}
	return r
}

func (e *Elections) String() string {
	/* FIXME replace state Usage
	str := fmt.Sprintf("eee %10s %s  dbht %d\n", e.State.GetFactomNodeName(), e.Name, e.DBHeight)
	str += fmt.Sprintf("eee %10s  %s\n", e.State.GetFactomNodeName(), "Federated Servers")
	for _, s := range e.Federated {
		str += fmt.Sprintf("eee %10s     %x\n", e.State.GetFactomNodeName(), s.GetChainID().Bytes())
	}
	str += fmt.Sprintf("eee %10s  %s\n", e.State.GetFactomNodeName(), "Audit Servers")
	for _, s := range e.Audit {
		str += fmt.Sprintf("eee %10s     %x\n", e.State.GetFactomNodeName(), s.GetChainID().Bytes())
	}
	return str
	*/
	return ""
}

func (e *Elections) SetElections3() {
	/* FIXME replace state Usage
	e.State.GetState().Election3 = fmt.Sprintf("%3s %15s %15s\n", "#", "Federated", "Audit")
	for i := 0; i < len(e.Federated)+len(e.Audit); i++ {
		fed := ""
		aud := ""
		if i < len(e.Federated) {
			id := e.Federated[i].GetChainID()
			fed = id.String()[6:12]
		}
		if i < len(e.Audit) {
			id := e.Audit[i].GetChainID()
			aud = id.String()[6:12]
		}
		if fed == "" && aud == "" {
			break
		}
		e.State.GetState().Election3 += fmt.Sprintf("%3d %15s %15s\n", i, fed, aud)
	}
	*/

}

func (e *Elections) Print() {
	fmt.Println(e.String())
}

// Returns the index of the given server. -1 if it isn't a Federated Server
func (e *Elections) LeaderIndex(server interfaces.IHash) int {
	for i, b := range e.Federated {
		if server.IsSameAs(b.GetChainID()) {
			return i
		}
	}
	return -1
}

// Returns the index of the given server. -1 if it isn't a Audit Server
func (e *Elections) AuditIndex(server interfaces.IHash) int {
	for i, b := range e.Audit {
		if server.IsSameAs(b.GetChainID()) {
			return i
		}
	}

	return -1
}

func (e *Elections) AuditAdapterIndex(server interfaces.IHash) int {
	if e.Adapter == nil {
		return -1
	}
	for i, b := range e.Adapter.GetAudits() {
		if server.IsSameAs(b) {
			return i
		}
	}
	return -1
}

func (e *Elections) AuditPriority() int {
	// Get the priority order list of audit servers in the priority order
	for len(e.Round) <= e.Electing {
		e.Round = append(e.Round, 0)
	}
	e.APriority = Order(e.Audit, e.DBHeight, e.Minute, e.Electing)
	auditIdx := MaxIdx(e.APriority)
	return auditIdx
}

func (e *Elections) debugExec() (ret bool) {
	return globals.Params.DebugLogRegEx != ""
}

func (e *Elections) LogPrintLeaders(log string) {
	e.LogPrintf(log, "%6s | %6s", "Fed", "Aud")
	limit := len(e.Federated)
	if limit < len(e.Audit) {
		limit = len(e.Audit)
	}
	for i := 0; i < limit; i++ {
		f := ""
		a := ""
		if i < len(e.Federated) {
			f = fmt.Sprintf("%6x", e.Federated[i].GetChainID().Bytes()[3:6])
		}
		if i < len(e.Audit) {
			a = fmt.Sprintf("%6x", e.Audit[i].GetChainID().Bytes()[3:6])
		}
		e.LogPrintf(log, "%6s | %6s", f, a)
	}

}

func (e *Elections) LogPrintf(logName string, format string, more ...interface{}) {
	/* FIXME get rid of state
	s := e.State.GetState()
	if e.debugExec() {
		s.LogPrintf(logName, format, more...)
	}
	*/
}

func (e *Elections) LogMessage(logName string, comment string, msg interfaces.IMsg) {
	/* FIXME get rid of state
	s := e.State.GetState()
	if e.debugExec() {
		s.LogMessage(logName, comment, msg)
	}
	*/
}

// Check that the process list and Election Authority Sets match
func (e *Elections) CheckAuthSetsMatch(caller string) {

	pl := e.state.(*state.State).ProcessLists.Get(uint32(e.DBHeight))
	var s_fservers, s_aservers []interfaces.IServer
	if pl == nil {
		s_fservers = make([]interfaces.IServer, 0)
		s_aservers = make([]interfaces.IServer, 0)
	} else {
		s_fservers = pl.FedServers
		s_aservers = pl.AuditServers
	}

	e_fservers := e.Federated
	e_aservers := e.Audit

	printAll := func(format string, more ...interface{}) {
		fmt.Printf(e.state.GetFactomNodeName()+":"+caller+":"+format+"\n", more...)
		e.LogPrintf("election", caller+":"+format, more...)
		e.state.LogPrintf("executeMsg", caller+":"+format, more...)
	}

	var dummy state.Server = state.Server{primitives.ZeroHash, "dummy", false, primitives.ZeroHash}

	// Force the lists to be the same size by adding Dummy
	for len(s_fservers) > len(e_fservers) {
		e_fservers = append(e_fservers, &dummy)
	}

	for len(s_fservers) < len(e_fservers) {
		s_fservers = append(s_fservers, &dummy)
	}

	for len(s_aservers) > len(e_aservers) {
		e_aservers = append(e_aservers, &dummy)
	}

	for len(s_aservers) < len(e_aservers) {
		s_aservers = append(s_aservers, &dummy)
	}

	var mismatch1 bool
	for i, f := range s_fservers {
		if e_fservers[i].GetChainID() != f.GetChainID() {
			printAll("Process List FedSet is not the same as Election FedSet at %d", i)
			mismatch1 = true
		}
	}
	if mismatch1 {
		printAll("Federated %d", len(s_fservers))
		printAll("idx election process")
		for i, _ := range s_fservers {
			printAll("%3d  %x  %x", i, e_fservers[i].GetChainID().Bytes()[3:6], s_fservers[i].GetChainID().Bytes()[3:6])
		}
		printAll("")
	}

	var mismatch2 bool
	for i, f := range s_aservers {
		if e_aservers[i].GetChainID() != f.GetChainID() {
			printAll("Process List AudSet is not the same as Election AudSet at %d", i)
			mismatch2 = true
		}
	}
	if mismatch2 {
		printAll("Audit %d", len(s_aservers))
		printAll("idx election process")
		for i, _ := range s_aservers {
			printAll("%3d  %x  %x", i, e_aservers[i].GetChainID().Bytes()[3:6], s_aservers[i].GetChainID().Bytes()[3:6])
		}
		printAll("")
	}

	//if !mismatch1 && !mismatch2 {
	//	printAll("AuthSet Matched!")
	//}
}

func (e *Elections) GetIgnoreMissing() bool {
	return e.state.(*state.State).IgnoreMissing
}

func (e *Elections) GetDBFinished() bool {
	return e.state.(*state.State).GetDBFinished()
}

func (e *Elections) GetIdentityChainID() interfaces.IHash {
	return e.state.(*state.State).GetIdentityChainID()
}

func (e *Elections) Sign(b []byte) interfaces.IFullSignature {
	return e.state.(*state.State).ServerPrivKey.Sign(b)
}

func (e *Elections) GetState() *state.State {
	return e.state.(*state.State)
}

func (e *Elections) Enqueue(msg interfaces.IMsg) {
	e.Input.Enqueue(msg)
}
