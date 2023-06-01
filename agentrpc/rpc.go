package agentrpc

//type FrameOfInterest struct {
//	// !!!
//	//Gid   int
//	//Frame int
//	Value []string
//}

type Snapshot struct {
	Stacks map[int]string
	// Map from goroutine ID to map from frame index to array of captured values.
	// The frame indexes match the order in Stacks - from leaf function to
	// callers.
	Frames_of_interest map[int]map[int][]string
	// !!! Frames_of_interest map[int]map[int]FrameOfInterest
}

type GetSnapshotIn struct{}
type GetSnapshotOut struct {
	Snapshot Snapshot
}
