package poset

import (
	"fmt"

	"github.com/golang/protobuf/proto"
)

/*
Roots constitute the base of a Poset. Each Participant is assigned a Root on
top of which Events will be added. The first Event of a participant must have a
Self-Parent and an Other-Parent that match its Root X and Y respectively.

This construction allows us to initialize Posets where the first Events are
taken from the middle of another Poset

ex 1:

-----------------        -----------------       -----------------
- Event E0      -        - Event E1      -       - Event E2      -
- SP = ""       -        - SP = ""       -       - SP = ""       -
- OP = ""       -        - OP = ""       -       - OP = ""       -
-----------------        -----------------       -----------------
        |                        |                       |
-----------------		 -----------------		 -----------------
- Root 0        - 		 - Root 1        - 		 - Root 2        -
- X = Y = ""    - 		 - X = Y = ""    -		 - X = Y = ""    -
- Index= -1     -		 - Index= -1     -       - Index= -1     -
- Others= empty - 		 - Others= empty -       - Others= empty -
-----------------		 -----------------       -----------------

ex 2:

-----------------
- Event E02     -
- SP = E01      -
- OP = E_OLD    -
-----------------
       |
-----------------
- Event E01     -
- SP = E00      -
- OP = E10      -  \
-----------------    \
       |               \
-----------------        -----------------       -----------------
- Event E00     -        - Event E10     -       - Event E20     -
- SP = x0       -        - SP = x1       -       - SP = x2       -
- OP = y0       -        - OP = y1       -       - OP = y2       -
-----------------        -----------------       -----------------
        |                        |                       |
-----------------		 -----------------		 -----------------
- Root 0        - 		 - Root 1        - 		 - Root 2        -
- X: x0, Y: y0  - 		 - X: x1, Y: y1  - 		 - X: x2, Y: y2  -
- Index= i0     -		 - Index= i1     -       - Index= i2     -
- Others= {     - 		 - Others= empty -       - Others= empty -
-  E02: E_OLD   -        -----------------       -----------------
- }             -
-----------------
*/

// RootEvent contains enough information about an Event and its direct descendant
// to allow inserting Events on top of it.
// NewBaseRootEvent creates a RootEvent corresponding to the the very beginning
// of a Poset.
func NewBaseRootEvent(creatorID int64) RootEvent {
	hash := fmt.Sprintf("Root%d", creatorID)
	res := RootEvent{
		Hash:             hash,
		CreatorID:        creatorID,
		Index:            -1,
		LamportTimestamp: -1,
		Round:            -1,
	}
	return res
}

func (m *RootEvent) Equals(that *RootEvent) bool {
	return m.Hash == that.Hash &&
		m.CreatorID == that.CreatorID &&
		m.Index == that.Index &&
		m.LamportTimestamp == that.LamportTimestamp &&
		m.Round == that.Round
}

// Root forms a base on top of which a participant's Events can be inserted. It
// contains the SelfParent of the first descendant of the Root, as well as other
// Events, belonging to a past before the Root, which might be referenced
// in future Events. NextRound corresponds to a proposed value for the child's
// Round; it is only used if the child's OtherParent is empty or NOT in the
// Root's Others.
// NewBaseRoot initializes a Root object for a fresh Poset.
func NewBaseRoot(creatorID int64) Root {
	rootEvent := NewBaseRootEvent(creatorID)
	res := Root{
		NextRound:  0,
		SelfParent: &rootEvent,
		Others:     map[string]*RootEvents{},
	}
	return res
}

// TODO: Fix it
func EqualsMapStringRootEvent(this map[string]*RootEvents, that map[string]*RootEvents) bool {
	if len(this) != len(that) {
		return false
	}
	for k, v := range this {
		v2, ok := that[k]
		if !ok || len(v.Value) != len(v2.Value) {
			return false
		}
	}
	return true
}

func (m *Root) Equals(that *Root) bool {
	return m.NextRound == that.NextRound &&
		m.SelfParent.Equals(that.SelfParent) &&
		EqualsMapStringRootEvent(m.Others, that.Others)
}

func (m *Root) ProtoMarshal() ([]byte, error) {
	var bf proto.Buffer
	bf.SetDeterministic(true)
	if err := bf.Marshal(m); err != nil {
		return nil, err
	}
	return bf.Bytes(), nil
}

func (m *Root) ProtoUnmarshal(data []byte) error {
	return proto.Unmarshal(data, m)
}
