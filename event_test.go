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
	"crypto/ecdsa"
	"reflect"
	"testing"
	"time"
)

func createDummyEventBody() EventBody {
	body := EventBody{}
	body.Transactions = [][]byte{[]byte("abc"), []byte("def")}
	body.Parents = [][]byte{[]byte("self"), []byte("other")}
	body.Creator = ecdsa.PublicKey{}
	body.Timestamp = time.Now()
	return body
}

func TestMarshallBody(t *testing.T) {
	body := createDummyEventBody()

	raw, err := body.Marshal()
	if err != nil {
		t.Fatalf("Error marshalling EventBody: %s", err)
	}

	newBody := new(EventBody)
	if err := newBody.Unmarshal(raw); err != nil {
		t.Fatalf("Error unmarshalling EventBody: %s", err)
	}

	if !reflect.DeepEqual(body.Transactions, newBody.Transactions) {
		t.Fatalf("Payloads do not match. Expected %#v, got %#v", body.Transactions, newBody.Transactions)
	}
	if !reflect.DeepEqual(body.Parents, newBody.Parents) {
		t.Fatalf("Parents do not match. Expected %#v, got %#v", body.Parents, newBody.Parents)
	}
	if !reflect.DeepEqual(body.Creator, newBody.Creator) {
		t.Fatalf("Creators do not match. Expected %#v, got %#v", body.Creator, newBody.Creator)
	}
	if body.Timestamp != newBody.Timestamp {
		t.Fatalf("Timestamps do not match. Expected %#v, got %#v", body.Timestamp, newBody.Timestamp)
	}

}
