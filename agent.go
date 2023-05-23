package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/kr/pretty"
	"log"
	"strconv"
)

var addrFlag = flag.String("addr", "127.0.0.1:45689", "")

var script = `
frames_of_interest = {
  # 'executeWriteBatch': 'ba',
  'executeRead': 'ba',
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
			# {"FollowPointers":True, "MaxVariableRecurse":1, "MaxStringLen":0, "MaxArrayValues":0, "MaxStructFields":100}, # MaxVariableRecurse:1, MaxStringLen:64, MaxArrayValues:64, MaxStructFields:-1}"
			)
		# Search for frames of interest.

		i = 0
		for f in stack.Locations:
			for foi in frames_of_interest:
				if not f.Location.Function:
					continue
				# print("looking at ", g.ID, i, f.Location.Function.Name_)
				if foi in f.Location.Function.Name_:
					print("found ", g.ID, i, f.Location.Function.Name_)
					res.append((g.ID, i, foi, f.Location.Function.Name_))
			i = i+1
		# print('-----------------------')

	print("res: ", res)
	vars = []
	for r in res:
		print({"GoroutineID": r[0], "Frame": r[1]}) # , frames_of_interest[r[2]])
		vars.append(eval(
				{"GoroutineID": r[0], "Frame": r[1]}, 
				frames_of_interest[r[2]],
				{"FollowPointers":True, "MaxVariableRecurse":3, "MaxStringLen":100, "MaxArrayValues":10, "MaxStructFields":100}
			).Variable)

	print("looked at #goroutines: ", len(gs))
	return json.encode(vars)

def main():
	print("hello world")
	return gs()
`

func main() {
	flag.Parse()
	client := rpc2.NewClient(*addrFlag)

	state, err := client.Halt()
	if err != nil {
		panic(err)
	}
	log.Printf("got state: running:%t", state.Running)

	out, err := client.ExecScript(script)
	if err != nil {
		log.Printf("executing script failed: %s\n.Output:%s", err, out.Output)
	} else {
		unquoted, err := strconv.Unquote(out.Val)
		if err != nil {
			panic(err)
		}
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, []byte(unquoted), "", "\t"); err != nil {
			panic(err)
		}
		log.Printf("script output: %sval: (%s)", out.Output, prettyJSON.String())
	}

	ch := client.Continue()
	for state := range ch {
		log.Printf("got state: %s", pretty.Sprint(state))
	}
	log.Print("finished with continue; channel closed")
}
