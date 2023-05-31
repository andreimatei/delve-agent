package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/kr/pretty"
	"log"
	"os"
	"strconv"
	"time"
)

var addrFlag = flag.String("addr", "127.0.0.1:45689", "")

type FrameOfInterest struct {
	Gid   int
	Frame int
	Value string
}

type Snapshot struct {
	Stacks             map[int]string
	Frames_of_interest []FrameOfInterest
}

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

		starScript, err := os.ReadFile("walk_stacks.star")
		if err != nil {
			log.Fatal(err)
		}
		out, err := client.ExecScript(string(starScript))
		if err != nil {
			log.Printf("executing script failed: %s\nOutput:%s", err, out.Output)
		} else {
			unquoted, err := strconv.Unquote(out.Val)
			if err != nil {
				panic(err)
			}
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, []byte(unquoted), "", "\t"); err != nil {
				panic(err)
			}
			// log.Printf("script output: %sval: %s", out.Output, prettyJSON.String())

			var s Snapshot
			err = json.Unmarshal([]byte(unquoted), &s)
			if err != nil {
				log.Printf("%v. failed to decode: %s", err, out.Val)
				panic(err)
			}
			pretty.Print(s)
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
