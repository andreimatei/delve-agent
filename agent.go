package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/go-delve/delve/service/rpc2"
	"log"
	"strconv"
	"strings"
	"time"
)

var addrFlag = flag.String("addr", "127.0.0.1:45689", "")

var script = `
frames_of_interest = {
	# 'executeWriteBatch': 'ba',
	#'executeRead': 'ba',
	# 'execStmtInOpenState': 'stmt.SQL'
	'execStmtInOpenState': 'parserStmt.SQL'
	# 'executeRead': 'ba.Requests[0].Value.(*kvpb.RequestUnion_Get).Get'
}

def gs():
	gs = goroutines().Goroutines

	res = []

	for g in gs:
		stack = stacktrace(g.ID,
			100,   # depth
			False, # full
			False, # defers
			7,     # option flags
			# {"FollowPointers":True, "MaxVariableRecurse":3, "MaxStringLen":0, "MaxArrayValues":10, "MaxStructFields":100}, # MaxVariableRecurse:1, MaxStringLen:64, MaxArrayValues:64, MaxStructFields:-1}"
			)

		# Search for frames of interest.
		i = 0
		for f in stack.Locations:
			for foi in frames_of_interest:
				if not f.Location.Function:
					continue
				# print("looking at ", g.ID, i, f.Location.Function.Name_, f.Location.File, f.Location.File.Line)
				if foi in f.Location.Function.Name_:
					print("found frame of interest: gid: %d:%d, func: %s, location: %s:%d (0x{%d})" %
						(g.ID, i, f.Location.Function.Name_, f.Location.File, f.Location.Line, f.Location.PC))
					res.append((g.ID, i, foi, f.Location.Function.Name_))
			i = i+1
		# print('-----------------------')

	print("res: ", res)
	vars = []
	for r in res:
		(gid, frame, foi, loc) = r
		print("reading from GoroutineID: %d, Frame: %d, foi: %s loc: %s" % (gid, frame, foi, loc)) # , frames_of_interest[r[2]])
		vars.append(eval(
				{"GoroutineID": r[0], "Frame": r[1]}, 
				frames_of_interest[r[2]],
				# panicky config: {"FollowPointers":True, "MaxVariableRecurse":2, "MaxStringLen":100, "MaxArrayValues":10, "MaxStructFields":100}
				{"FollowPointers":True, "MaxVariableRecurse":2, "MaxStringLen":100, "MaxArrayValues":10, "MaxStructFields":100}
			).Variable.Value)  # .Value
		print("reading succeed")

	print("looked at #goroutines: ", len(gs))
	return json.encode(vars)

def main():
	return gs()
`

func main() {
	flag.Parse()
	client := rpc2.NewClient(*addrFlag)

	for {
		_ /* state */, err := client.Halt()
		if err != nil {
			panic(err)
		}
		// log.Printf("got state: running:%t", state.Running)

		out, err := client.ExecScript(script)
		if err != nil {
			log.Printf("executing script failed: %s\nOutput:%s", err, out.Output)
		} else {
			unquoted, err := strconv.Unquote(out.Val)
			if err != nil {
				panic(err)
			}
			if strings.Contains(unquoted, "val: ([])\n") {
				// !!!
			}
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, []byte(unquoted), "", "\t"); err != nil {
				panic(err)
			}
			log.Printf("script output: %sval: (%s)", out.Output, prettyJSON.String())
		}

		go func() {
			ch := client.Continue()
			_ = ch
			//for state := range ch {
			//	log.Printf("got state: %s", pretty.Sprint(state))
			//}
			//log.Print("finished with continue; channel closed")
		}()
		time.Sleep(100 * time.Millisecond)
	}
}
