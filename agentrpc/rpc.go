package agentrpc

import "github.com/go-delve/delve/pkg/proc"

// input and output of RPCs. In a separate package because they're shared with
// client services.

//// Copied from Delve API pkg.
//type VarInfo struct {
//	Name    string
//	Type    TypeInfo
//	VarType int
//}
//
//type TypeInfo struct {
//	Name   string
//	Fields []FieldInfo
//}
//
//type FieldInfo struct {
//	Name string
//	Type TypeInfo
//}

type Snapshot struct {
	Stacks map[int]string
	// Map from goroutine ID to map from frame index to array of captured values.
	// The frame indexes match the order in Stacks - from leaf function to
	// callers.
	Frames_of_interest map[int]map[int][]string
}

type GetSnapshotIn struct {
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
	Vars  []proc.VarInfo
	Types []proc.TypeInfo
}
