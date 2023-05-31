package agentrpc

type FrameOfInterest struct {
	Gid   int
	Frame int
	Value string
}

type Snapshot struct {
	Stacks             map[int]string
	Frames_of_interest []FrameOfInterest
}

type GetSnapshotIn struct{}
type GetSnapshotOut struct {
	Snapshot Snapshot
}
