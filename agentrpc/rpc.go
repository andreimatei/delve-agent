package agentrpc

import (
	"github.com/go-delve/delve/service/debugger"
)

// input and output of RPCs. In a separate package because they're shared with
// client services.

type Snapshot struct {
	Stacks map[int]string
	// Map from goroutine ID to map from frame index to array of captured values.
	// The frame indexes match the order in Stacks - from leaf function to
	// callers.
	FramesOfInterest map[int]map[int][]CapturedExpr
	// FlightRecorder is a dump of the recorded data. The recorded data consists
	// of a map from key to buffer representing the latest events with that key.
	FlightRecorder map[string][]string
}

type CapturedExpr struct {
	Expr string
	Val  string
}

type GetSnapshotIn struct {
	// FramesSpec maps from function name to list of expressions to evaluate and
	// collect.
	FramesSpec map[string][]string
}

type GetSnapshotOut struct {
	Snapshot Snapshot
}

type ListVarsIn struct {
	Func  string
	PCOff int64
}
type ListVarsOut struct {
	Vars  []debugger.VarInfo
	Types []debugger.TypeInfo
}

type GetTypeInfoIn struct {
	Name string
}
type GetTypeInfoOut struct {
	Fields []FieldInfo
}
type FieldInfo struct {
	Name     string
	TypeName string
	Embedded bool
}

type FlightRecorderEventSpec struct {
	Frame   string
	Expr    string
	KeyExpr string
}

type ReconcileFlightRecorderIn struct {
	Events []FlightRecorderEventSpec
}

type ReconcileFLightRecorderOut struct {
}
