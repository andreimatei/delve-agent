package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/pprof/profile"
	"github.com/kr/pretty"
	pp "github.com/maruel/panicparse/v2/stack"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andreimatei/delve-agent/agentrpc"
	"github.com/go-delve/delve/service/rpc2"
)

var delveAddrFlag = flag.String("addr", "127.0.0.1:45689", "")
var grpclistenAddrFlag = flag.String("listen-grpc", "127.0.0.1:1235", "")
var oneShot = flag.Bool("oneshot", false, "")

type grpcServer struct {
	agentrpc.UnsafeDebugInfoServer
	agentrpc.UnsafeSnapshotServiceServer
	client *rpc2.RPCClient
}

func (s *grpcServer) DownloadBinary(ctx context.Context, in *agentrpc.DownloadBinaryIn) (*agentrpc.DownloadBinaryOut, error) {
	//TODO implement me
	panic("implement me")
}

func (s *grpcServer) ListProcesses(in *agentrpc.ListProcessesIn, server agentrpc.DebugInfo_ListProcessesServer) error {
	//TODO implement me
	panic("implement me")
}

var _ agentrpc.DebugInfoServer = &grpcServer{}
var _ agentrpc.SnapshotServiceServer = &grpcServer{}

func (s *grpcServer) haltTarget() (resume func()) {
	_ /* state */, err := s.client.Halt()
	if err != nil {
		panic(err)
	}
	return s.continueTarget
}

func (s *grpcServer) continueTarget() {
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

func (s *grpcServer) ListFunctions(ctx context.Context, args *agentrpc.ListFunctionsIn) (*agentrpc.ListFunctionsOut, error) {
	defer s.haltTarget()()

	funcs, err := s.client.ListFunctions(args.Filter)
	if err != nil {
		return nil, err
	}
	if args.Limit > 0 && len(funcs) > int(args.Limit) {
		funcs = funcs[:args.Limit]
	}
	return &agentrpc.ListFunctionsOut{Funcs: funcs}, nil
}

func (s *grpcServer) ListTypes(ctx context.Context, args *agentrpc.ListTypesIn) (*agentrpc.ListTypesOut, error) {
	defer s.haltTarget()()

	types, err := s.client.ListTypes(args.Filter)
	if err != nil {
		return nil, err
	}
	if args.Limit > 0 && len(types) > int(args.Limit) {
		types = types[:args.Limit]
	}
	return &agentrpc.ListTypesOut{Types: types}, nil
}

func (s *grpcServer) GetTypeInfo(ctx context.Context, in *agentrpc.GetTypeInfoIn) (*agentrpc.GetTypeInfoOut, error) {
	fmt.Printf("GetTypeInfo...")
	// Halt the target and defer the resumption.
	defer s.haltTarget()()

	typ, err := s.client.GetTypeInfo(in.TypeName)
	if err != nil {
		return nil, err
	}

	fieldsList := make([]*agentrpc.FieldInfo, len(typ.Fields))
	for j, f := range typ.Fields {
		fieldsList[j] = &agentrpc.FieldInfo{
			FieldName: f.Name,
			TypeName:  f.TypeName,
			Embedded:  f.Embedded,
		}
	}

	out := &agentrpc.GetTypeInfoOut{
		Fields: fieldsList,
	}
	return out, nil
}

func (s *grpcServer) ListVars(ctx context.Context, in *agentrpc.ListVarsIn) (*agentrpc.ListVarsOut, error) {
	// Halt the target and defer the resumption.
	defer s.haltTarget()()

	vars, types, err := s.client.ListAvailableVariables(in.FuncName, in.PcOffset, int(in.TypeRecursionLimit), -1 /* maxTypes */, 10 /* maxFieldsPerStruct */)
	if err != nil {
		return nil, err
	}

	// Convert from the Delve response to our format.
	varsList := make([]*agentrpc.VarInfo, len(vars))
	for i, v := range vars {
		varsList[i] = &agentrpc.VarInfo{
			VarName:          v.Name,
			TypeName:         v.Type,
			FormalParameter:  v.FormalParameter,
			LoclistAvailable: v.LoclistAvailable,
		}
	}
	typesMap := make(map[string]*agentrpc.TypeInfo, len(types))
	for _, typ := range types {
		fieldsList := make([]*agentrpc.FieldInfo, len(typ.Fields))
		for j, f := range typ.Fields {
			fieldsList[j] = &agentrpc.FieldInfo{
				FieldName: f.Name,
				TypeName:  f.TypeName,
				Embedded:  f.Embedded,
			}
		}
		typesMap[typ.Name] = &agentrpc.TypeInfo{
			Name: typ.Name,
			// TODO(andrei): We're not actually getting the HasFields value from
			// Delve, so we're approximating it.
			HasFields:       typ.FieldsNotLoaded || len(typ.Fields) > 0,
			Fields:          fieldsList,
			FieldsNotLoaded: typ.FieldsNotLoaded,
		}
	}

	return &agentrpc.ListVarsOut{
		Vars:  varsList,
		Types: typesMap,
	}, nil
}

// !!!
//func (s *grpcServer) GetSnapshot(ctx context.Context, in *agentrpc.GetSnapshotIn) (*agentrpc.GetSnapshotOut, error) {
//	// Halt the target and defer the resumption.
//	defer s.haltTarget()()
//
//	starScript, err := os.ReadFile("walk_stacks.star")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Parameterize the script with the frames of interest.
//	var sb strings.Builder
//	for frame, exprs := range in.FramesSpec {
//		sb.WriteString(fmt.Sprintf("'%s': [", frame))
//		for i, expr := range exprs {
//			if i > 0 {
//				sb.WriteString(", ")
//			}
//			sb.WriteString(fmt.Sprintf("'%s'", expr))
//		}
//		sb.WriteString("],\n")
//	}
//	// Run the script.
//	script := strings.Replace(string(starScript), "$frames_spec", sb.String(), 1)
//	typeSpecs := typeSpecsToStarlark(in.TypeSpecs)
//	log.Printf("typeSpecs: %s", typeSpecs)
//	script = strings.Replace(script, "$type_specs", typeSpecsToStarlark(in.TypeSpecs), 1)
//
//	scriptRes, err := s.client.ExecScript(script)
//	if err != nil {
//		return nil, fmt.Errorf("executing script failed: %w\nOutput:%s", err, scriptRes.Output)
//	}
//	unquoted, err := strconv.Unquote(scriptRes.Val)
//	if err != nil {
//		panic(err)
//	}
//	// Unmarshal the script results.
//	var snap scriptResults
//	err = json.Unmarshal([]byte(unquoted), &snap)
//	if err != nil {
//		log.Printf("%v. failed to decode: %s", err, unquoted)
//		panic(err)
//	}
//
//	// Read the flight recorder data and attach it to the results.
//	frData, err := s.client.GetFlightRecorderData()
//	if err != nil {
//		return nil, err
//	}
//
//	out.Snapshot = agentrpc.Snapshot{
//		Stacks:             snap.Stacks,
//		FramesOfInterest:   snap.FramesOfInterest,
//		FlightRecorderData: frData.Data,
//	}
//	return nil
//}

func scriptResultsToPProf(stacks map[int]string) (*profile.Profile, error) {
	stacksStr := stacksToString(stacks)
	// Parse the stacks.
	opts := pp.DefaultOpts()
	opts.ParsePC = true
	snap, _, err := pp.ScanSnapshot(strings.NewReader(stacksStr), io.Discard, opts)
	if err != io.EOF {
		return nil, fmt.Errorf("failed to scan stacks: %w", err)
	}
	agg := snap.Aggregate(pp.AnyValue)
	b := newPProfBuilder()

	for _, group := range agg.Buckets {
		b.addSample(group.Signature.Stack.Calls, group.IDs)
	}
	b.profile.TimeNanos = time.Now().UnixNano()
	return b.profile, nil
}

//type Server struct {
//	client *rpc2.RPCClient
//}
//
//func (s *Server) continueProcess() {
//	// Continue blocks, so we do it on a different goroutine that leaks.
//	go func() {
//		ch := s.client.Continue()
//		_ = ch
//		//for state := range ch {
//		//	log.Printf("got state: %s", pretty.Sprint(state))
//		//}
//		//log.Print("finished with continue; channel closed")
//	}()
//}
//
//func (s *Server) ReconcileFlightRecorder(in agentrpc.ReconcileFlightRecorderIn, out *agentrpc.ReconcileFLightRecorderOut) error {
//	log.Printf("!!! ReconcileFlightRecorder: %v", in)
//	_ /* state */, err := s.client.Halt()
//	if err != nil {
//		panic(err)
//	}
//	defer s.continueProcess()
//
//	scriptTemplate := `
//stmt = eval(None, "$expr")
//flight_recorder(str(cur_scope().GoroutineID), stmt.Variable.Value)
//`
//
//	bks, err := s.client.ListBreakpoints(false /* all ? */)
//	if err != nil {
//		return err
//	}
//
//	evName := func(ev agentrpc.FlightRecorderEventSpec) string {
//		return fmt.Sprintf("%s-%s", ev.Frame, ev.Expr)
//	}
//
//	findEv := func(name string) int {
//		for i, ev := range in.Events {
//			if name == evName(ev) {
//				return i
//			}
//		}
//		return -1
//	}
//
//	// map from idx
//	alreadyExists := make(map[int]struct{})
//	for _, bk := range bks {
//		evIdx := findEv(bk.Name)
//		if evIdx == -1 {
//			_, err := s.client.ClearBreakpoint(bk.ID)
//			if err != nil {
//				return nil
//			}
//		}
//		log.Printf("event %s already exists", bk.Name)
//		alreadyExists[evIdx] = struct{}{}
//	}
//
//	for i, ev := range in.Events {
//		if _, ok := alreadyExists[i]; ok {
//			continue
//		}
//
//		keyExpr := ev.KeyExpr
//		if ev.KeyExpr == "goroutineID" {
//			keyExpr = `str(cur_scope().GoroutineID)`
//		}
//		script := strings.ReplaceAll(scriptTemplate, "$expr", ev.Expr)
//		script = strings.ReplaceAll(script, "$keyExpr", keyExpr)
//		fmt.Printf("script: %s\n", script)
//
//		locs, err := s.client.FindLocation(api.EvalScope{
//			GoroutineID:  -1,
//			Frame:        0,
//			DeferredCall: 0,
//		},
//			ev.Frame,
//			true, // findInstructions
//			nil,  // substitutePathRules
//		)
//		if err != nil {
//			return err
//		}
//		if len(locs) != 1 {
//			return fmt.Errorf("found %d locations for %s", len(locs), ev.Frame)
//		}
//
//		log.Printf("creating breakpoint: %s - %s (%s)", ev.Frame, ev.Expr, ev.KeyExpr)
//		_, err = s.client.CreateBreakpoint(&api.Breakpoint{
//			Name:   evName(ev),
//			Addrs:  []uint64{locs[0].PC},
//			File:   locs[0].File,
//			Line:   locs[0].Line,
//			Script: script,
//		})
//		if err != nil {
//			return err
//		}
//		log.Printf("installed breakpoint: %s - %s (%s)", ev.Frame, ev.Expr, ev.KeyExpr)
//	}
//
//	return nil
//}

type CapturedExpr struct {
	Expr string
	Val  string
}

// scriptResults is the result of running the walk_stacks.star script.
type scriptResults struct {
	Stacks map[int]string `json:"stacks"`
	// Map from goroutine ID to map from frame index to array of captured values.
	// The frame indexes match the order in Stacks - from leaf function to
	// callers.
	FramesOfInterest map[int]map[int][]CapturedExpr `json:"frames_of_interest"`
}

// GetSnapshot collects the stack traces of all the goroutines and the requested
// data for the specified frames of interest.
func (s *grpcServer) GetSnapshot(ctx context.Context, in *agentrpc.GetSnapshotIn) (*agentrpc.GetSnapshotOut, error) {
	log.Printf("!!! GetSnapshot...")
	// Halt the target and defer the resumption.
	defer s.haltTarget()()

	starScript, err := os.ReadFile("walk_stacks.star")
	if err != nil {
		log.Fatal(err)
	}

	// Parameterize the script with the frames of interest.
	var sb strings.Builder
	for _, frameSpec := range in.FrameSpecs {
		sb.WriteString(fmt.Sprintf("'%s': [", frameSpec.FuncName))
		for i, expr := range frameSpec.Expressions {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("'%s'", expr))
		}
		sb.WriteString("],\n")
	}
	// Run the script.
	script := strings.Replace(string(starScript), "$frames_spec", sb.String(), 1)
	typeSpecs := typeSpecsToStarlark(in.TypeSpecs)
	log.Printf("typeSpecs: %s", typeSpecs)
	script = strings.Replace(script, "$type_specs", typeSpecsToStarlark(in.TypeSpecs), 1)

	scriptRes, err := s.client.ExecScript(script)
	if err != nil {
		log.Printf("script failed: %v\nOutput:%s", err, scriptRes.Output)
		return nil, fmt.Errorf("executing script failed: %w\nOutput:%s", err, scriptRes.Output)
	}

	log.Printf("!!! script output: %s", scriptRes.Output)

	unquoted, err := strconv.Unquote(scriptRes.Val)
	if err != nil {
		panic(err)
	}
	// Unmarshal the script results.
	var snap scriptResults
	err = json.Unmarshal([]byte(unquoted), &snap)
	if err != nil {
		log.Printf("%v. failed to decode: %s", err, unquoted)
		panic(err)
	}
	profile, err := scriptResultsToPProf(snap.Stacks)
	if err != nil {
		return nil, fmt.Errorf("failed to parse script results: %w", err)
	}

	var frameData []*agentrpc.FrameData
	for gid, fois := range snap.FramesOfInterest {
		for frameIdx, capturedExprs := range fois {
			var data []*agentrpc.CapturedExpression
			for _, v := range capturedExprs {
				data = append(data, &agentrpc.CapturedExpression{
					Expression: v.Expr,
					Value:      v.Val,
				})
			}
			frameData = append(frameData, &agentrpc.FrameData{
				GoroutineId:   int64(gid),
				FrameIdx:      int64(frameIdx),
				CapturedExprs: data,
			})
		}
	}

	//// Read the flight recorder data and attach it to the results.
	//frData, err := s.client.GetFlightRecorderData()
	//if err != nil {
	//	return nil, err
	//}

	return &agentrpc.GetSnapshotOut{
		Profile:   profileToProto(profile),
		FrameData: frameData,
		// !!!
		//FlightRecorderData: frData.Data,
	}, nil
}

func typeSpecsToStarlark(specs []*agentrpc.TypeSpec) string {
	// Generate a starlark list looking list this:
	//	typeSpecs := `[
	//	{
	//	"TypeName": "github.com/cockroachdb/cockroach/pkg/kv/kvpb.PutRequest",
	//	"CollectAll: false,
	//	"LoadSpec": {"Exprs": ["Value", "Inline", "Blind"]},
	//	},
	//  ...
	//]`

	var sb strings.Builder
	sb.WriteString("[\n")
	for _, spec := range specs {
		sb.WriteString("{\n")
		sb.WriteString(fmt.Sprintf("\t\"TypeName\": \"%s\",\n", spec.TypeName))
		collectAllStr := "False"
		if spec.CollectAll {
			collectAllStr = "True"
		}
		sb.WriteString(fmt.Sprintf("\t\"LoadSpec\": {\"CollectAll\": %s, \"Exprs\": [", collectAllStr))
		for i, expr := range spec.Expressions {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("\"%s\"", expr))
		}
		sb.WriteString("]},\n}\n")
	}
	sb.WriteString("]")
	return sb.String()
}

func stacksToString(stacks map[int]string) string {
	var sb strings.Builder
	// We ignore the goroutine ID map key; the stacks themselves start by
	// identifying the goroutine.
	for _, stack := range stacks {
		sb.WriteString(stack)
		sb.WriteRune('\n')
	}
	return sb.String()
}

func main() {
	flag.Parse()

	client := rpc2.NewClient(*delveAddrFlag)

	if *oneShot {
		gs, _, err := client.ListGoroutines(0, 10000)
		if err != nil {
			panic(err)
		}
		for _, g := range gs {
			stack, err := client.StacktraceEx(g.ID, 500, 0, nil)
			if err != nil {
				panic(err)
			}
			pretty.Print(stack)
		}

		return
	}

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

	grpcSrv := grpc.NewServer()
	serverImpl := &grpcServer{client: client}
	agentrpc.RegisterDebugInfoServer(grpcSrv, serverImpl)
	agentrpc.RegisterSnapshotServiceServer(grpcSrv, serverImpl)

	l, e := net.Listen("tcp", *grpclistenAddrFlag)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	log.Printf("Serving gRPC on %s", *grpclistenAddrFlag)
	_ = grpcSrv.Serve(l)
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
