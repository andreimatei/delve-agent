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

def serialize_backtrace(gid):
	stack = stacktrace(gid,
		100,   # depth
		False, # full
		False, # defers
		# 7,     # option flags
		)
	backtrace = ''
	for i, f in enumerate(stack.Locations):
		fun_name = '<unknown>'
		if f.Location.Function:
			fun_name = f.Location.Function.Name_
		backtrace = backtrace + '%d - %s %s:%d (0x%x)\n' % (i, fun_name, f.Location.File, f.Location.Line, f.Location.PC)
	return backtrace

def gs():
	gs = goroutines().Goroutines

	res = []

	stacks = {}
	for g in gs:
		stack = stacktrace(g.ID,
			100,   # depth
			False, # full
			False, # defers
			# 7,     # option flags
			# {"FollowPointers":True, "MaxVariableRecurse":3, "MaxStringLen":0, "MaxArrayValues":10, "MaxStructFields":100}, # MaxVariableRecurse:1, MaxStringLen:64, MaxArrayValues:64, MaxStructFields:-1}"
			)

		# Search for frames of interest.
		backtrace = ''
		i = 0
		for f in stack.Locations:
			fun_name = '<unknown>'
			if f.Location.Function:
				fun_name = f.Location.Function.Name_
			backtrace = backtrace + '%d:%d - %s %s:%d (0x%x)\n' % (g.ID, i, fun_name, f.Location.File, f.Location.Line, f.Location.PC)
			for foi in frames_of_interest:
				if not f.Location.Function:
					continue
				if f.Location.Function.Name_.endswith(foi):
					print("found frame of interest: gid: %d:%d, func: %s, location: %s:%d (0x%x)" %
						(g.ID, i, f.Location.Function.Name_, f.Location.File, f.Location.Line, f.Location.PC))
					res.append((g.ID, i, foi, f.Location.Function.Name_))
			i = i+1
		stacks[g.ID] = backtrace

	if res:
		print('-----------------------')
	for r in res:
		(gid, frame, foi, loc) = r
		print(stacks[gid])

	print("res: ", res)
	vars = []
	for r in res:
		(gid, frame, foi, loc) = r
		print("reading from GoroutineID: %d, Frame: %d, foi: %s loc: %s" % (gid, frame, foi, loc)) # , frames_of_interest[r[2]])
		backtrace = serialize_backtrace(gid)
		print("backtrace for %d: %s" % (gid, backtrace))
		vars.append(eval(
				{"GoroutineID": gid, "Frame": frame}, 
				frames_of_interest[foi],
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
		time.Sleep(time.Second)

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
		time.Sleep(time.Second)
		//time.Sleep(100 * time.Millisecond)
	}
}
