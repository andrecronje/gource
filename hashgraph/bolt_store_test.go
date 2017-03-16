/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package hashgraph

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/arrivets/babble/crypto"
)

func initBoltStore() (*BoltStore, []pub) {
	n := 3
	participantPubs := []pub{}
	participants := []string{}
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateECDSAKey()
		pubKey := crypto.FromECDSAPub(&key.PublicKey)
		participantPubs = append(participantPubs,
			pub{pubKey, fmt.Sprintf("0x%X", pubKey)})
		participants = append(participants, fmt.Sprintf("0x%X", pubKey))
	}

	store, err := NewBoltStore("test.db", participants)
	if err != nil {
		fmt.Println("ERROR creating BoltStore")
	}
	return store, participantPubs
}

func closeBoltStore(s *BoltStore) {
	s.Close()
	os.Remove(s.fn)
}

func TestBoltEvents(t *testing.T) {
	store, participants := initBoltStore()
	defer closeBoltStore(store)

	events := make(map[string]Event)
	for _, p := range participants {
		event := NewEvent([][]byte{[]byte("abc")}, []string{"", ""}, p.pubKey)
		events[p.hex] = event
		err := store.SetEvent(event)
		if err != nil {
			t.Fatal(err)
		}
	}

	for p, ev := range events {
		rev, err := store.GetEvent(ev.Hex())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ev, rev) {
			t.Fatalf("Stored Event from %s... does not match", p[:10])
		}
	}

	for _, p := range participants {
		pEvents, err := store.ParticipantEvents(p.hex)
		if err != nil {
			t.Fatal(err)
		}
		if l := len(pEvents); l != 1 {
			t.Fatalf("%s should have 1 event, not %d", p, l)
		}
		expectedEvent := events[p.hex]
		if pEvents[0] != expectedEvent.Hex() {
			t.Fatalf("%s ParticipantEvents do not match", p)
		}
	}

	expectedKnow := make(map[string]int)
	for _, p := range participants {
		expectedKnow[p.hex] = 1
	}
	known := store.Known()
	if !reflect.DeepEqual(expectedKnow, known) {
		t.Fatalf("Incorrect Known. Got %#v, expected %#v", known, expectedKnow)
	}

	for _, p := range participants {
		e := events[p.hex]
		if err := store.AddConsensusEvent(e.Hex()); err != nil {
			t.Fatal(err)
		}
	}
	consensusEvents := store.ConsensusEvents()
	for i, p := range participants {
		e := events[p.hex]
		if c := consensusEvents[i]; c != e.Hex() {
			t.Fatalf("ConsensusEvents[%d] should be %s..., not %s...", i, e.Hex()[:10], c[:10])
		}
	}
}

func TestBoltRounds(t *testing.T) {
	store, participants := initBoltStore()
	defer closeBoltStore(store)

	round := NewRoundInfo()
	events := make(map[string]Event)
	for _, p := range participants {
		event := NewEvent([][]byte{}, []string{"", ""}, p.pubKey)
		events[p.hex] = event
		round.AddEvent(event.Hex(), true)
	}

	if err := store.SetRound(0, *round); err != nil {
		t.Fatal(err)
	}

	if c := store.Rounds(); c != 1 {
		t.Fatalf("Store should count 1 round, not %d", c)
	}

	storedRound, err := store.GetRound(0)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(*round, storedRound) {
		t.Fatalf("Round and StoredRound do not match")
	}

	witnesses := store.RoundWitnesses(0)
	expectedWitnesses := round.Witnesses()
	if len(witnesses) != len(expectedWitnesses) {
		t.Fatalf("There should be %d witnesses, not %d", len(expectedWitnesses), len(witnesses))
	}
	for _, w := range expectedWitnesses {
		if !contains(witnesses, w) {
			t.Fatalf("Witnesses should contain %s", w)
		}
	}
}
