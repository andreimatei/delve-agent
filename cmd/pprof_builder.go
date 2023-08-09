package main

import (
	"encoding/binary"
	"github.com/andreimatei/delve-agent/agentrpc"
	pp "github.com/maruel/panicparse/v2/stack"
	"hash/fnv"
)

type locationKey struct {
	functionName string
	pcOffset     int64
}

type pprofBuilder struct {
	// profile is the profile being populated.
	profile *agentrpc.Profile
	// functionMap keeps track of all functions in profile, mapping the function
	// name to location ID.
	functionMap map[string]uint64
	// locationMap keeps track of all locations in profile, mapping (function
	// name, pc offset) to the location ID.
	locationMap map[locationKey]uint64
	// stringsMap keeps track of all strings in profile, mapping the string to its
	// index in the strings table.
	stringsMap map[string]int64
}

func newPProfBuilder() *pprofBuilder {
	return &pprofBuilder{
		profile: &agentrpc.Profile{
			SampleType: nil,
			Sample:     nil,
			Mapping:    nil,
			Location:   nil,
			Function:   nil,
			// By pprof spec, the empty string is always present at index 0.
			StringTable:       []string{""},
			DropFrames:        0,
			KeepFrames:        0,
			TimeNanos:         0,
			DurationNanos:     0,
			PeriodType:        nil,
			Period:            0,
			Comment:           nil,
			DefaultSampleType: 0,
		},
		functionMap: make(map[string]uint64),
		locationMap: make(map[locationKey]uint64),
		stringsMap:  map[string]int64{"": 0},
	}
}

func (b *pprofBuilder) addSample(locIDs []uint64, gIDs []int) {
	labelIdx := b.getOrAddString(agentrpc.GoroutineIDLabel)
	var labels []*agentrpc.Label
	for _, gID := range gIDs {
		labels = append(labels, &agentrpc.Label{
			Key:     labelIdx,
			Num:     int64(gID),
			NumUnit: labelIdx, // We use the name of the label as the unit too.
		})
	}

	sample := &agentrpc.Sample{
		LocationId: locIDs,
		Value:      nil,
		Label:      labels,
	}
	b.profile.Sample = append(b.profile.Sample, sample)
}

// Returns the ID of the location.
func (b *pprofBuilder) getOrAddLocation(call pp.Call) uint64 {
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

	location := &agentrpc.Location{
		Id:        hash,
		MappingId: 0,    // TODO
		Address:   hash, // HACK
		Line: []*agentrpc.Line{{
			FunctionId: b.getOrAddFunction(call),
			Line:       int64(call.Line),
		}},
		IsFolded: false,
	}
	b.profile.Location = append(b.profile.Location, location)
	b.locationMap[locKey] = location.Id
	return location.Id
}

func (b *pprofBuilder) getOrAddFunction(call pp.Call) uint64 {
	funcName := call.Func.Name
	h := fnv.New64()
	h.Write([]byte(funcName))
	hash := h.Sum64()

	if id, ok := b.functionMap[funcName]; ok {
		return id
	}
	function := &agentrpc.Function{
		Id:         hash,
		Name:       b.getOrAddString(call.Func.Name),
		SystemName: b.getOrAddString(""),
		Filename:   b.getOrAddString(call.RemoteSrcPath),
		StartLine:  0,
	}
	b.profile.Function = append(b.profile.Function, function)
	b.functionMap[funcName] = function.Id
	return function.Id
}

func (b *pprofBuilder) getOrAddString(s string) int64 {
	if id, ok := b.stringsMap[s]; ok {
		return id
	}
	b.stringsMap[s] = int64(len(b.profile.StringTable))
	b.profile.StringTable = append(b.profile.StringTable, s)
	return b.stringsMap[s]
}
