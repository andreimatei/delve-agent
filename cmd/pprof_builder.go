package main

import (
	"bytes"
	"encoding/binary"
	"github.com/andreimatei/delve-agent/agentrpc"
	"github.com/google/pprof/profile"
	pp "github.com/maruel/panicparse/v2/stack"
	"google.golang.org/protobuf/proto"
	"hash/fnv"
)

type locationKey struct {
	functionName string
	pcOffset     int64
}

type pprofBuilder struct {
	// profile is the profile being populated.
	profile *profile.Profile
	// functionMap keeps track of all functions in profile, mapping the function
	// name to *Function.
	functionMap map[string]*profile.Function
	// locationMap keeps track of all locations in profile, mapping (function
	// name, pc offset) to *Location.
	locationMap map[locationKey]*profile.Location
}

func newPProfBuilder() *pprofBuilder {
	return &pprofBuilder{
		profile: &profile.Profile{
			Mapping: []*profile.Mapping{
				{
					ID:              1,
					BuildID:         "dummy-build-id",
					Start:           0x0,
					Limit:           ^uint64(0),
					HasFunctions:    true,
					HasFilenames:    true,
					HasLineNumbers:  true,
					HasInlineFrames: true,
				},
			},
		},
		functionMap: make(map[string]*profile.Function),
		locationMap: make(map[locationKey]*profile.Location),
	}
}

func (b *pprofBuilder) addSample(calls []pp.Call, gIDs []int) {
	labels := map[string][]int64{
		agentrpc.GoroutineIDLabel: make([]int64, len(gIDs)),
	}
	for i, gID := range gIDs {
		labels[agentrpc.GoroutineIDLabel][i] = int64(gID)
	}

	var locs []*profile.Location
	for _, call := range calls {
		locs = append(locs, b.getOrAddLocation(call))
	}

	sample := &profile.Sample{
		Location: locs,
		Value:    nil,
		NumLabel: labels,
	}
	b.profile.Sample = append(b.profile.Sample, sample)
}

// Returns the ID of the location.
func (b *pprofBuilder) getOrAddLocation(call pp.Call) *profile.Location {
	// Compute the hash of the function name and PC offset.
	h := fnv.New64()
	h.Write([]byte(call.Func.Name))
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(call.PCOffset))
	h.Write(buf[:])
	hash := h.Sum64()

	locKey := locationKey{
		functionName: call.Func.Name,
		pcOffset:     call.PCOffset,
	}
	if id, ok := b.locationMap[locKey]; ok {
		return id
	}

	location := &profile.Location{
		ID:      hash,
		Mapping: b.profile.Mapping[0],
		Address: hash, // HACK
		Line: []profile.Line{{
			Function: b.getOrAddFunction(call),
			Line:     int64(call.Line),
		}},
		IsFolded: false,
	}
	b.profile.Location = append(b.profile.Location, location)
	b.locationMap[locKey] = location
	return location
}

func (b *pprofBuilder) getOrAddFunction(call pp.Call) *profile.Function {
	funcName := call.Func.Name
	h := fnv.New64()
	h.Write([]byte(funcName))
	hash := h.Sum64()

	if id, ok := b.functionMap[funcName]; ok {
		return id
	}
	function := &profile.Function{
		ID:         hash,
		Name:       call.Func.Name,
		SystemName: "",
		Filename:   call.RemoteSrcPath,
		StartLine:  0,
	}
	b.profile.Function = append(b.profile.Function, function)
	b.functionMap[funcName] = function
	return function
}

func profileToProto(p *profile.Profile) *agentrpc.Profile {
	var buf bytes.Buffer
	err := p.WriteUncompressed(&buf)
	if err != nil {
		panic(err)
	}
	var pb agentrpc.Profile
	err = proto.Unmarshal(buf.Bytes(), &pb)
	if err != nil {
		panic(err)
	}
	return &pb
}
