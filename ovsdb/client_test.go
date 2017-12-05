// Copyright 2017 DigitalOcean.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ovsdb_test

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/digitalocean/go-openvswitch/ovsdb"
	"github.com/digitalocean/go-openvswitch/ovsdb/internal/jsonrpc"
	"github.com/google/go-cmp/cmp"
)

func TestClientJSONRPCError(t *testing.T) {
	const str = "some error"

	c, _, done := testClient(t, func(_ jsonrpc.Request) jsonrpc.Response {
		return jsonrpc.Response{
			ID:    intPtr(1),
			Error: str,
		}
	})
	defer done()

	_, err := c.ListDatabases()
	if err == nil {
		t.Fatal("expected an error, but none occurred")
	}
}

func TestClientOVSDBError(t *testing.T) {
	const str = "some error"

	c, _, done := testClient(t, func(_ jsonrpc.Request) jsonrpc.Response {
		return jsonrpc.Response{
			ID: intPtr(1),
			Result: mustMarshalJSON(t, &ovsdb.Error{
				Err:     str,
				Details: "malformed",
				Syntax:  "{}",
			}),
		}
	})
	defer done()

	_, err := c.ListDatabases()
	if err == nil {
		t.Fatal("expected an error, but none occurred")
	}

	oerr, ok := err.(*ovsdb.Error)
	if !ok {
		t.Fatalf("error of wrong type: %#v", err)
	}

	if diff := cmp.Diff(str, oerr.Err); diff != "" {
		t.Fatalf("unexpected error (-want +got):\n%s", diff)
	}
}

func TestClientListDatabases(t *testing.T) {
	want := []string{"Open_vSwitch", "test"}

	c, _, done := testClient(t, func(req jsonrpc.Request) jsonrpc.Response {
		if diff := cmp.Diff("list_dbs", req.Method); diff != "" {
			panicf("unexpected RPC method (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(0, len(req.Params)); diff != "" {
			panicf("unexpected number of RPC parameters (-want +got):\n%s", diff)
		}

		return jsonrpc.Response{
			ID:     intPtr(1),
			Result: mustMarshalJSON(t, want),
		}
	})
	defer done()

	dbs, err := c.ListDatabases()
	if err != nil {
		t.Fatalf("failed to list databases: %v", err)
	}

	if diff := cmp.Diff(want, dbs); diff != "" {
		t.Fatalf("unexpected databases (-want +got):\n%s", diff)
	}
}

func testClient(t *testing.T, fn jsonrpc.TestFunc) (*ovsdb.Client, chan<- *jsonrpc.Response, func()) {
	t.Helper()

	conn, notifC, done := jsonrpc.TestNetConn(t, fn)

	c, err := ovsdb.New(conn, ovsdb.Debug(log.New(os.Stderr, "", 0)))
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	return c, notifC, func() {
		_ = c.Close()
		done()
	}
}

func mustMarshalJSON(t *testing.T, v interface{}) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	return b
}

func intPtr(i int) *int {
	return &i
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
