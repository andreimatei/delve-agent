frames_of_interest = {
    # 'executeWriteBatch': 'ba',
    #'executeRead': 'ba',
    # 'execStmtInOpenState': 'stmt.SQL'
    'execStmtInOpenState': 'parserStmt.SQL'
    # 'executeRead': 'ba.Requests[0].Value.(*kvpb.RequestUnion_Get).Get'
}


goroutine_status_to_string = {
    0: "idle",
    1: "runnable",
    2: "running",
    3: "syscall",
    4: "waiting",
    # 5: "moribund",  # supposedly unused
    6: "dead",
    7: "enqueue",
    8: "copystack",
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

    g_out = {}
    for g in gs:
        stack = stacktrace(g.ID,
                           100,   # depth
                           False, # full
                           False, # defers
                           # 7,     # option flags
                           # {"FollowPointers":True, "MaxVariableRecurse":3, "MaxStringLen":0, "MaxArrayValues":10, "MaxStructFields":100}, # MaxVariableRecurse:1, MaxStringLen:64, MaxArrayValues:64, MaxStructFields:-1}"
                           )


        # Search for frames of interest.
        backtrace = 'goroutine %d [%s]:\n' % (g.ID, goroutine_status_to_string[g.Status])
        i = 0
        for f in stack.Locations:
            fun_name = '<unknown>'
            if f.Location.Function:
                fun_name = f.Location.Function.Name_
            backtrace = backtrace + '%s\n\t%s:%d\n' % (fun_name, f.Location.File, f.Location.Line)
            for foi in frames_of_interest:
                if not f.Location.Function:
                    continue
                if f.Location.Function.Name_.endswith(foi):
                    # print("found frame of interest: gid: %d:%d, func: %s, location: %s:%d (0x%x)" %
                    #	 (g.ID, i, f.Location.Function.Name_, f.Location.File, f.Location.Line, f.Location.PC))
                    res.append((g.ID, i, foi, f.Location.Function.Name_))
            i = i+1
        # print(backtrace)
        g_out[g.ID] = backtrace
        
        # if len(g_out) == 3:
        #     break

    # if res:
    # 	print('-----------------------')
    # for r in res:
    # 	(gid, frame, foi, loc) = r
    # 	print(stacks[gid])

    # print("res: ", res)
    vars = []
    for r in res:
        (gid, frame, foi, loc) = r
        #print("reading from GoroutineID: %d, Frame: %d, foi: %s loc: %s" % (gid, frame, foi, loc)) # , frames_of_interest[r[2]])
        #backtrace = serialize_backtrace(gid)
        #print("backtrace for %d: %s" % (gid, backtrace))
        val = eval(
            {"GoroutineID": gid, "Frame": frame},
            frames_of_interest[foi],
            {"FollowPointers":True, "MaxVariableRecurse":2, "MaxStringLen":100, "MaxArrayValues":10, "MaxStructFields":100}
        ).Variable.Value
        vars.append(struct(gid = gid, frame = frame, value = val))

    print("looked at #goroutines: ", len(gs))
    res = {
        "stacks": g_out,
        "frames_of_interest": vars,
    }
    return json.encode(res)

def main():
    return gs()
