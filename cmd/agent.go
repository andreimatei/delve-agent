package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-delve/delve/service/api"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"

	"github.com/andreimatei/delve-agent/agentrpc"
	"github.com/go-delve/delve/service/rpc2"
)

var delveAddrFlag = flag.String("addr", "127.0.0.1:45689", "")
var listenAddrFlag = flag.String("listen", "127.0.0.1:1234", "")

type Server struct {
	client *rpc2.RPCClient
}

func (s *Server) continueProcess() {
	// Continue blocks, so we do it on a different goroutine that leaks.
	go func() {
		ch := s.client.Continue()
		_ = ch
		//for state := range ch {
		//	log.Printf("got state: %s", pretty.Sprint(state))
		//}
		//log.Print("finished with continue; channel closed")
	}()
}

// scriptResults is the result of running the walk_stacks.star script.
type scriptResults struct {
	Stacks map[int]string `json:"stacks"`
	// Map from goroutine ID to map from frame index to array of captured values.
	// The frame indexes match the order in Stacks - from leaf function to
	// callers.
	FramesOfInterest map[int]map[int][]agentrpc.CapturedExpr `json:"frames_of_interest"`
}

// GetSnapshot collects the stack traces of all the goroutines and the requested
// data for the specified frames of interest.
func (s *Server) GetSnapshot(in agentrpc.GetSnapshotIn, out *agentrpc.GetSnapshotOut) error {
	log.Printf("!!! GetSnapshot")
	_ /* state */, err := s.client.Halt()
	if err != nil {
		panic(err)
	}
	defer s.continueProcess()

	starScript, err := os.ReadFile("walk_stacks.star")
	if err != nil {
		log.Fatal(err)
	}

	// Parameterize the script with the frames of interest.
	var sb strings.Builder
	for frame, exprs := range in.FramesSpec {
		sb.WriteString(fmt.Sprintf("'%s': [", frame))
		for i, expr := range exprs {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("'%s'", expr))
		}
		sb.WriteString("],\n")
	}
	// Run the script.
	log.Printf("!!! GetSnapshot: running script with args: %s", sb.String())
	script := strings.Replace(string(starScript), "$frames_spec", sb.String(), 1)
	scriptRes, err := s.client.ExecScript(script)
	if err != nil {
		return fmt.Errorf("executing script failed: %w\nOutput:%s", err, scriptRes.Output)
	}
	unquoted, err := strconv.Unquote(scriptRes.Val)
	if err != nil {
		panic(err)
	}
	log.Printf("!!! GetSnapshot: script result: %s", unquoted)
	// Unmarshal the script results.
	var snap scriptResults
	err = json.Unmarshal([]byte(unquoted), &snap)
	if err != nil {
		log.Printf("%v. failed to decode: %s", err, unquoted)
		panic(err)
	}

	// Read the flight recorder data and attach it to the results.
	frData, err := s.client.GetFlightRecorderData()
	if err != nil {
		return err
	}

	log.Printf("!!! GetSnapshot: stacks: %+v\nframes of interest:%s", snap.Stacks, snap.FramesOfInterest)
	out.Snapshot = agentrpc.Snapshot{
		Stacks:             snap.Stacks,
		FramesOfInterest:   snap.FramesOfInterest,
		FlightRecorderData: frData.Data,
	}
	log.Printf("!!! GetSnapshot: done. Frames of interest: %+v", out.Snapshot.FramesOfInterest)
	return nil
}

func (s *Server) ListVars(in agentrpc.ListVarsIn, out *agentrpc.ListVarsOut) error {
	log.Printf("!!! ListVars...")

	// !!! The halt is necessary, otherwise the RPC below blocks. Why?
	_ /* state */, err := s.client.Halt()
	if err != nil {
		panic(err)
	}
	defer s.continueProcess()

	vars, types, err := s.client.ListAvailableVariables(in.Func, in.PCOff, 3 /* typeLevels */, -1 /* maxTypes */, 10 /* maxFieldsPerStruct */)
	if err != nil {
		log.Printf("!!! ListVars... err: %s", err)
		return err
	}
	out.Vars = vars
	out.Types = types
	//for _, t := range types {
	//	log.Printf("!!! got type: %s loaded: %t", t.Name, !t.FieldsNotLoaded)
	//}
	//out.Vars = make([]agentrpc.VarInfo, len(vars))
	//for i, v := range vars {
	//	out.Vars[i] = agentrpc.VarInfo{
	//		Name:    v.Name,
	//		VarType: v.VarType,
	//		Type:    convertType(v.),
	//	}
	//}
	return nil
}

func (s *Server) GetTypeInfo(in agentrpc.GetTypeInfoIn, out *agentrpc.GetTypeInfoOut) error {
	log.Printf("!!! GetTypeInfo: %s", in.Name)

	_ /* state */, err := s.client.Halt()
	if err != nil {
		panic(err)
	}
	defer s.continueProcess()

	typ, err := s.client.GetTypeInfo(in.Name)
	if err != nil {
		log.Printf("!!! GetTypeInfo... err: %s", err)
		return err
	}
	log.Printf("!!! response: %+v", typ)
	out.Fields = make([]agentrpc.FieldInfo, len(typ.Fields))
	for i, f := range typ.Fields {
		out.Fields[i] = agentrpc.FieldInfo{
			Name:     f.Name,
			TypeName: f.TypeName,
			Embedded: f.Embedded,
		}
	}
	return nil
}

func (s *Server) ReconcileFlightRecorder(in agentrpc.ReconcileFlightRecorderIn, out *agentrpc.ReconcileFLightRecorderOut) error {
	log.Printf("!!! ReconcileFlightRecorder: %v", in)
	_ /* state */, err := s.client.Halt()
	if err != nil {
		panic(err)
	}
	defer s.continueProcess()

	scriptTemplate := `
stmt = eval(None, "$expr")
flight_recorder(str(cur_scope().GoroutineID), stmt.Variable.Value)
`

	bks, err := s.client.ListBreakpoints(false /* all ? */)
	if err != nil {
		return err
	}

	evName := func(ev agentrpc.FlightRecorderEventSpec) string {
		return fmt.Sprintf("%s-%s", ev.Frame, ev.Expr)
	}

	findEv := func(name string) int {
		for i, ev := range in.Events {
			if name == evName(ev) {
				return i
			}
		}
		return -1
	}

	// map from idx
	alreadyExists := make(map[int]struct{})
	for _, bk := range bks {
		evIdx := findEv(bk.Name)
		if evIdx == -1 {
			_, err := s.client.ClearBreakpoint(bk.ID)
			if err != nil {
				return nil
			}
		}
		log.Printf("event %s already exists", bk.Name)
		alreadyExists[evIdx] = struct{}{}
	}

	for i, ev := range in.Events {
		if _, ok := alreadyExists[i]; ok {
			continue
		}

		keyExpr := ev.KeyExpr
		if ev.KeyExpr == "goroutineID" {
			keyExpr = `str(cur_scope().GoroutineID)`
		}
		script := strings.ReplaceAll(scriptTemplate, "$expr", ev.Expr)
		script = strings.ReplaceAll(script, "$keyExpr", keyExpr)
		fmt.Printf("script: %s\n", script)

		locs, err := s.client.FindLocation(api.EvalScope{
			GoroutineID:  -1,
			Frame:        0,
			DeferredCall: 0,
		},
			ev.Frame,
			true, // findInstructions
			nil,  // substitutePathRules
		)
		if err != nil {
			return err
		}
		if len(locs) != 1 {
			return fmt.Errorf("found %d locations for %s", len(locs), ev.Frame)
		}

		log.Printf("creating breakpoint: %s - %s (%s)", ev.Frame, ev.Expr, ev.KeyExpr)
		_, err = s.client.CreateBreakpoint(&api.Breakpoint{
			Name:   evName(ev),
			Addrs:  []uint64{locs[0].PC},
			File:   locs[0].File,
			Line:   locs[0].Line,
			Script: script,
		})
		if err != nil {
			return err
		}
		log.Printf("installed breakpoint: %s - %s (%s)", ev.Frame, ev.Expr, ev.KeyExpr)
	}

	return nil
}

func main() {
	flag.Parse()

	client := rpc2.NewClient(*delveAddrFlag)

	//starScript, err := os.ReadFile("query_break.star")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//bp, err := client.CreateBreakpoint(&api.Breakpoint{
	//	Name:         "test",
	//	File:         "/home/andrei/src/github.com/cockroachdb/cockroach/pkg/sql/conn_executor_exec.go",
	//	Line:         276,
	//	FunctionName: "",
	//	Cond:         "",
	//	Script:       string(starScript),
	//})
	//if err != nil {
	//	panic(err)
	//}
	//pretty.Print(bp)
	//ch := client.Continue()
	//for s := range ch {
	//	pretty.Print(s)
	//}

	srv := &Server{client: client}
	if err := rpc.RegisterName("Agent", srv); err != nil {
		panic(err)
	}
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", *listenAddrFlag)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	_ = http.Serve(l, nil)
}

//func parseSnapshot(s Snapshot) (*pp.Snapshot, error) {
//	var sb strings.Builder
//	for _, stack := range s.Stacks {
//		//log.Printf("!!! decoding: %s", stack)
//		//unq, err := strconv.Unquote(stack)
//		//if err != nil {
//		//	panic(err)
//		//}
//		sb.WriteString(stack)
//		sb.WriteRune('\n')
//	}
//	snapS := sb.String()
//	//log.Printf("!!! will parse:\b%s", snapS)
//
//	snap, _, err := pp.ScanSnapshot(strings.NewReader(snapS), io.Discard, pp.DefaultOpts())
//	if err != nil && err != io.EOF {
//		log.Printf("!!! failed to parse: %s:\n%s", err, snapS)
//		panic(err)
//		return nil, err
//	}
//	return snap, nil
//}
