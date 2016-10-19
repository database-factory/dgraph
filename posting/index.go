/*
 * Copyright 2016 Dgraph Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 		http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package posting

import (
	"context"
	"time"

	"golang.org/x/net/trace"

	"github.com/dgraph-io/dgraph/geo"
	"github.com/dgraph-io/dgraph/index"
	"github.com/dgraph-io/dgraph/posting/types"
	"github.com/dgraph-io/dgraph/schema"
	"github.com/dgraph-io/dgraph/store"
	stype "github.com/dgraph-io/dgraph/types"
	"github.com/dgraph-io/dgraph/x"
)

var (
	indexLog   trace.EventLog
	indexStore *store.Store
)

func init() {
	indexLog = trace.NewEventLog("index", "Logger")
}

// InitIndex initializes the index with the given data store.
func InitIndex(ds *store.Store) {
	if ds == nil {
		return
	}
	indexStore = ds
}

func tokenizedIndexKeys(attr string, p stype.Value) ([][]byte, error) {
	schemaType := schema.TypeOf(attr)
	if !schemaType.IsScalar() {
		return nil, x.Errorf("Cannot index attribute %s of type object.", attr)
	}
	s := schemaType.(stype.Scalar)
	schemaVal, err := s.Convert(p)
	if err != nil {
		return nil, err
	}
	if attr == "dob" {
		x.Printf("~~~~~tokenizedIndexKeys: %v %s %v", schema.TypeOf(attr), attr, schemaVal)
	}
	switch v := schemaVal.(type) {
	case *stype.Geo:
		return geo.IndexKeys(v)
	case *stype.Int32:
		return stype.IntIndex(attr, v)
	case *stype.Float:
		return stype.FloatIndex(attr, v)
	case *stype.Date:
		tmp, _ := stype.DateIndex(attr, v)
		x.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~~hey %s %v", attr, tmp)
		return stype.DateIndex(attr, v)
	case *stype.Time:
		return stype.TimeIndex(attr, v)
	case *stype.String:
		return stype.DefaultIndexKeys(attr, v), nil
	}
	return nil, nil
}

// spawnIndexMutations adds mutations to keep the index updated. The mutations
// may happen on a different worker.
func spawnIndexMutations(ctx context.Context, attr string, uid uint64, p stype.Value, del bool) {
	x.Assert(uid != 0)
	keys, err := tokenizedIndexKeys(attr, p)
	if err != nil {
		// This data is not indexable
		return
	}
	edge := x.DirectedEdge{
		Timestamp: time.Now(),
		ValueId:   uid,
		Attribute: attr,
		Source:    "idx",
	}

	for _, key := range keys {
		edge.Key = key
		if del {
			index.MutateChan <- x.Mutations{
				Del: []x.DirectedEdge{edge},
			}
			indexLog.Printf("DEL [%s] [%d] OldTerm [%s]", edge.Attribute, edge.Entity, string(key))
		} else {
			index.MutateChan <- x.Mutations{
				Set: []x.DirectedEdge{edge},
			}
			indexLog.Printf("SET [%s] [%d] NewTerm [%s]", edge.Attribute, edge.Entity, string(key))
		}
	}
}

// AddMutationWithIndex is AddMutation with support for indexing.
func (l *List) AddMutationWithIndex(ctx context.Context, t x.DirectedEdge, op byte) error {
	x.Assertf(len(t.Attribute) > 0 && t.Attribute[0] != ':',
		"[%s] [%d] [%s] %d %d\n", t.Attribute, t.Entity, string(t.Value), t.ValueId, op)

	var lastPost types.Posting
	var hasLastPost bool

	doUpdateIndex := indexStore != nil && (t.Value != nil) &&
		schema.IsIndexed(t.Attribute)
	if doUpdateIndex {
		// Check last posting for original value BEFORE any mutation actually happens.
		if l.Length() >= 1 {
			x.Assert(l.Get(&lastPost, l.Length()-1))
			hasLastPost = true
		}
	}
	hasMutated, err := l.AddMutation(ctx, t, op)
	if err != nil {
		return err
	}
	if !hasMutated || !doUpdateIndex {
		return nil
	}

	if hasLastPost && lastPost.ValueBytes() != nil {
		delTerm := lastPost.ValueBytes()
		delType := lastPost.ValType()
		p := stype.ValueForType(stype.TypeID(delType))
		err = p.UnmarshalBinary(delTerm)
		if err != nil {
			return err
		}
		spawnIndexMutations(ctx, t.Attribute, t.Entity, p, true)
	}
	if op == Set {
		p := stype.ValueForType(stype.TypeID(t.ValueType))
		err := p.UnmarshalBinary(t.Value)
		if err != nil {
			return err
		}
		spawnIndexMutations(ctx, t.Attribute, t.Entity, p, false)
	}
	return nil
}
