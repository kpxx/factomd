// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package messages

import (
	"bytes"
	"fmt"
	"github.com/FactomProject/factomd/common/interfaces"
	"github.com/FactomProject/factomd/common/primitives"
)

//A placeholder structure for messages
type CommitEntryMsg struct {
}

var _ interfaces.IMsg = (*CommitEntryMsg)(nil)

func (m *CommitEntryMsg) Type() int {
	return -1
}

func (m *CommitEntryMsg) Int() int {
	return -1
}

func (m *CommitEntryMsg) Bytes() []byte {
	return nil
}

func (m *CommitEntryMsg) UnmarshalBinaryData(data []byte) (newdata []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Error unmarshalling: %v", r)
		}
	}()

	return nil, nil
}

func (m *CommitEntryMsg) UnmarshalBinary(data []byte) error {
	_, err := m.UnmarshalBinaryData(data)
	return err
}

func (m *CommitEntryMsg) MarshalBinary() (data []byte, err error) {
	return nil, nil
}

func (m *CommitEntryMsg) String() string {
	return ""
}

func (m *CommitEntryMsg) DBHeight() int {
	return 0
}

func (m *CommitEntryMsg) ChainID() []byte {
	return nil
}

func (m *CommitEntryMsg) ListHeight() int {
	return 0
}

func (m *CommitEntryMsg) SerialHash() []byte {
	return nil
}

func (m *CommitEntryMsg) Signature() []byte {
	return nil
}

// Validate the message, given the state.  Three possible results:
//  < 0 -- Message is invalid.  Discard
//  0   -- Cannot tell if message is Valid
//  1   -- Message is valid
func (m *CommitEntryMsg) Validate(interfaces.IState) int {
	return 0
}

// Returns true if this is a message for this server to execute as
// a leader.
func (m *CommitEntryMsg) Leader(state interfaces.IState) bool {
	switch state.GetNetworkNumber() {
	case 0: // Main Network
		panic("Not implemented yet")
	case 1: // Test Network
		panic("Not implemented yet")
	case 2: // Local Network
		panic("Not implemented yet")
	default:
		panic("Not implemented yet")
	}

}

// Execute the leader functions of the given message
func (m *CommitEntryMsg) LeaderExecute(state interfaces.IState) error {
	return nil
}

// Returns true if this is a message for this server to execute as a follower
func (m *CommitEntryMsg) Follower(interfaces.IState) bool {
	return true
}

func (m *CommitEntryMsg) FollowerExecute(interfaces.IState) error {
	return nil
}

func (e *CommitEntryMsg) JSONByte() ([]byte, error) {
	return primitives.EncodeJSON(e)
}

func (e *CommitEntryMsg) JSONString() (string, error) {
	return primitives.EncodeJSONString(e)
}

func (e *CommitEntryMsg) JSONBuffer(b *bytes.Buffer) error {
	return primitives.EncodeJSONToBuffer(e, b)
}
