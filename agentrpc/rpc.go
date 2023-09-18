package agentrpc

const GoroutineIDLabel = "goroutine ID"

// input and output of RPCs. In a separate package because they're shared with
// client services.

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
