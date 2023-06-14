package main

import (
	"encoding/json"
	"flag"
	"fmt"
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

var addrFlag = flag.String("addr", "127.0.0.1:45689", "")

type Server struct {
	client *rpc2.RPCClient
}

func (s *Server) GetSnapshot(in agentrpc.GetSnapshotIn, out *agentrpc.GetSnapshotOut) error {
	log.Printf("!!! GetSnapshot")
	_ /* state */, err := s.client.Halt()
	if err != nil {
		panic(err)
	}

	defer func() {
		// Continue blocks, so we do it on a different goroutine that leaks.
		go func() {
			ch := s.client.Continue()
			_ = ch
			//for state := range ch {
			//	log.Printf("got state: %s", pretty.Sprint(state))
			//}
			//log.Print("finished with continue; channel closed")
		}()
	}()

	starScript, err := os.ReadFile("walk_stacks.star")
	if err != nil {
		log.Fatal(err)
	}

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
	log.Printf("!!! GetSnapshot: script: %s", sb.String())
	script := strings.Replace(string(starScript), "$frames_spec", sb.String(), 1)
	scriptRes, err := s.client.ExecScript(script)
	if err != nil {
		return fmt.Errorf("executing script failed: %w\nOutput:%s", err, scriptRes.Output)
	}

	unquoted, err := strconv.Unquote(scriptRes.Val)
	if err != nil {
		panic(err)
	}
	//var prettyJSON bytes.Buffer
	//if err := json.Indent(&prettyJSON, []byte(unquoted), "", "\t"); err != nil {
	//	panic(err)
	//}
	// log.Printf("script output: %sval: %s", out.Output, prettyJSON.String())

	var snap agentrpc.Snapshot
	err = json.Unmarshal([]byte(unquoted), &snap)
	if err != nil {
		log.Printf("%v. failed to decode: %s", err, unquoted)
		panic(err)
	}
	//ppSnap, err := parseSnapshot(snap)
	//if err != nil {
	//	panic(err)
	//}
	out.Snapshot = snap
	return nil
}

func (s *Server) ListVars(in agentrpc.ListVarsIn, out *agentrpc.ListVarsOut) error {
	log.Printf("!!! ListVars...")

	// !!! The halt is necessary, otherwise the RPC below blocks. Why?
	_ /* state */, err := s.client.Halt()
	if err != nil {
		panic(err)
	}

	vars, types, err := s.client.ListAvailableVariables(in.Func, in.PCOff)
	if err != nil {
		log.Printf("!!! ListVars... err: %s", err)
		return err
	}
	out.Vars = vars
	out.Types = types
	//out.Vars = make([]agentrpc.VarInfo, len(vars))
	//for i, v := range vars {
	//	out.Vars[i] = agentrpc.VarInfo{
	//		Name:    v.Name,
	//		VarType: v.VarType,
	//		Type:    convertType(v.),
	//	}
	//}
	log.Printf("!!! ListVars... res: %v", out.Vars)
	return nil
}

// !!!
//func convertType(info proc.TypeInfo) agentrpc.TypeInfo {
//	r := agentrpc.TypeInfo{
//		Name: info.Name,
//	}
//	r.Fields = make([]agentrpc.FieldInfo, len(info.Fields))
//	for i, f := range info.Fields {
//		r.Fields[i].Name = f.Name
//		r.Fields[i].Type = convertType(f.Type)
//	}
//	return r
//}

func main() {
	flag.Parse()

	client := rpc2.NewClient(*addrFlag)
	srv := &Server{client: client}

	if err := rpc.RegisterName("Agent", srv); err != nil {
		panic(err)
	}
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	_ = http.Serve(l, nil)

	//for {
	//	var snapOut GetSnapshotOut
	//	err := srv.GetSnapshot(GetSnapshotIn{}, &snapOut)
	//	if err != nil {
	//		panic(err)
	//	}
	//	//pretty.Print(snap)
	//
	//	ppSnap, err := parseSnapshot(snapOut.Snapshot)
	//	if err != nil {
	//		panic(err)
	//	}
	//	_ = ppSnap
	//	pretty.Print(ppSnap)
	//
	//	time.Sleep(time.Second)
	//}
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
