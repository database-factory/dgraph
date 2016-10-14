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

package query

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-geom"

	"github.com/dgraph-io/dgraph/geo"
	"github.com/dgraph-io/dgraph/posting"
	"github.com/dgraph-io/dgraph/schema"
	"github.com/dgraph-io/dgraph/store"
	"github.com/dgraph-io/dgraph/types"
	"github.com/dgraph-io/dgraph/worker"
	"github.com/dgraph-io/dgraph/x"
)

func createTestStore(t *testing.T) (string, *store.Store) {
	dir, err := ioutil.TempDir("", "storetest_")
	if err != nil {
		t.Error(err)
		return "", nil
	}

	ps, err := store.NewStore(dir)
	if err != nil {
		t.Error(err)
		return "", nil
	}

	worker.SetState(ps)

	posting.Init()
	schema.ParseBytes([]byte(`scalar geometry:geo @index`))
	posting.InitIndex(ps)
	return dir, ps
}

func addEdgeToStore(t *testing.T, edge x.DirectedEdge, ps *store.Store) {
	addEdge(t, edge, getOrCreate(posting.Key(edge.Entity, edge.Attribute), ps))
}

func addGeoData(t *testing.T, ps *store.Store, uid uint64, p geom.T, name string) {
	edge := x.DirectedEdge{
		Timestamp: time.Now(),
		Attribute: "geometry",
		ValueType: byte(types.GeoID),
		Entity:    uid,
	}
	g := types.Geo{p}
	var err error
	edge.Value, err = g.MarshalBinary()
	require.NoError(t, err)
	addEdgeToStore(t, edge, ps)

	edge.Value = []byte(name)
	edge.ValueType = byte(types.StringID)
	edge.Attribute = "name"
	addEdgeToStore(t, edge, ps)
}

func createTestData(t *testing.T, ps *store.Store) {
	p := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{-122.082506, 37.4249518})
	addGeoData(t, ps, 1, p, "Googleplex")

	p = geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{-122.080668, 37.426753})
	addGeoData(t, ps, 2, p, "Shoreline Amphitheater")

	p = geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{-122.2527428, 37.513653})
	addGeoData(t, ps, 3, p, "San Carlos Airport")

	poly := geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{
		{{-121.6, 37.1}, {-122.4, 37.3}, {-122.6, 37.8}, {-122.5, 38.3}, {-121.9, 38}, {-121.6, 37.1}},
	})
	addGeoData(t, ps, 4, poly, "SF Bay area")
	poly = geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{
		{{-122.06, 37.37}, {-122.1, 37.36}, {-122.12, 37.4}, {-122.11, 37.43}, {-122.04, 37.43}, {-122.06, 37.37}},
	})
	addGeoData(t, ps, 5, poly, "Mountain View")
	poly = geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{
		{{-122.25, 37.49}, {-122.28, 37.49}, {-122.27, 37.51}, {-122.25, 37.52}, {-122.24, 37.51}},
	})
	addGeoData(t, ps, 6, poly, "San Carlos")

	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.
}

func runQuery(t *testing.T, sg *SubGraph) string {
	ctx := context.Background()
	ch := make(chan error)
	go ProcessGraph(ctx, sg, nil, ch)
	err := <-ch
	require.NoError(t, err)

	var l Latency
	js, err := sg.ToJSON(&l)
	require.NoError(t, err)
	return string(js)
}

func TestWithinPoint(t *testing.T) {
	dir, ps := createTestStore(t)
	defer os.RemoveAll(dir)
	defer ps.Close()

	createTestData(t, ps)

	p := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{-122.082506, 37.4249518})
	g := types.Geo{p}
	data, err := g.MarshalBinary()
	require.NoError(t, err)

	sg := &SubGraph{
		Attr:      "geometry",
		GeoFilter: &geo.Filter{Data: data, Type: geo.QueryTypeWithin},
		Children:  []*SubGraph{&SubGraph{Attr: "name"}},
	}

	js := runQuery(t, sg)
	require.Equal(t, js, `{"name":"Googleplex"}`)
}

func TestWithinPolygon(t *testing.T) {
	dir, ps := createTestStore(t)
	defer os.RemoveAll(dir)
	defer ps.Close()

	createTestData(t, ps)

	p := geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{
		{{-122.06, 37.37}, {-122.1, 37.36}, {-122.12, 37.4}, {-122.11, 37.43}, {-122.04, 37.43}, {-122.06, 37.37}},
	})
	g := types.Geo{p}
	data, err := g.MarshalBinary()
	require.NoError(t, err)

	sg := &SubGraph{
		Attr:      "geometry",
		GeoFilter: &geo.Filter{Data: data, Type: geo.QueryTypeWithin},
		Children:  []*SubGraph{&SubGraph{Attr: "name"}},
	}

	js := runQuery(t, sg)
	require.Equal(t, js,
		`[{"name":"Googleplex"},{"name":"Shoreline Amphitheater"}]`)
}

func TestContainsPoint(t *testing.T) {
	dir, ps := createTestStore(t)
	defer os.RemoveAll(dir)
	defer ps.Close()

	createTestData(t, ps)

	p := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{-122.082506, 37.4249518})
	g := types.Geo{p}
	data, err := g.MarshalBinary()
	require.NoError(t, err)

	sg := &SubGraph{
		Attr:      "geometry",
		GeoFilter: &geo.Filter{Data: data, Type: geo.QueryTypeContains},
		Children:  []*SubGraph{&SubGraph{Attr: "name"}},
	}

	js := runQuery(t, sg)
	require.Equal(t, js,
		`[{"name":"SF Bay area"},{"name":"Mountain View"}]`)
}

func TestNearPoint(t *testing.T) {
	dir, ps := createTestStore(t)
	defer os.RemoveAll(dir)
	defer ps.Close()

	createTestData(t, ps)

	p := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{-122.082506, 37.4249518})
	g := types.Geo{p}
	data, err := g.MarshalBinary()
	require.NoError(t, err)

	sg := &SubGraph{
		Attr:      "geometry",
		GeoFilter: &geo.Filter{Data: data, Type: geo.QueryTypeNear, MaxDistance: 1000},
		Children:  []*SubGraph{&SubGraph{Attr: "name"}},
	}

	js := runQuery(t, sg)
	require.Equal(t, js,
		`[{"name":"Googleplex"},{"name":"Shoreline Amphitheater"}]`)
}
