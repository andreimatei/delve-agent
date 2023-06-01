# Maps from function name to list of expressions to evaluate in the scope of
# that function.
frames_of_interest = {
    'execStmtInOpenState': ['parserStmt.SQL', 'p.semaCtx.Placeholders.Values'],
    # 'executeWriteBatch': 'ba',
    # 'executeRead': 'ba',
    # 'executeRead': 'ba.Requests[0].Value.(*kvpb.RequestUnion_Get).Get'
}

goroutine_status_to_string = {
    0: "idle",
    1: "runnable",
    2: "running",
    3: "syscall",
    4: "waiting",
    5: "moribund",  # supposedly unused
    6: "dead",
    7: "enqueue",
    8: "copystack",
}


def serialize_backtrace(gid, limit):
    stack = stacktrace(gid,
                       100,  # depth
                       False,  # full
                       False,  # defers
                       # 7,     # option flags
                       )
    backtrace = ''
    for i, f in enumerate(stack.Locations):
        fun_name = '<unknown>'
        if f.Location.Function:
            fun_name = f.Location.Function.Name_
        backtrace = backtrace + '%d - %s %s:%d (0x%x)\n' % (
        i, fun_name, f.Location.File, f.Location.Line, f.Location.PC)
        if i == limit:
            break
    return backtrace


def gs():
    gs = goroutines().Goroutines

    # recognized_frames accumulates info about frames for which we'll evaluate
    # some expressions.
    recognized_frames = []
    g_out = {}
    for g in gs:
        stack = stacktrace(g.ID,
                           100,  # depth
                           False,  # full
                           False,  # defers
                           # 7,     # option flags
                           # {"FollowPointers":True, "MaxVariableRecurse":3, "MaxStringLen":0, "MaxArrayValues":10, "MaxStructFields":100}, # MaxVariableRecurse:1, MaxStringLen:64, MaxArrayValues:64, MaxStructFields:-1}"
                           )

        # Search for frames of interest.
        backtrace = 'goroutine %d [%s]:\n' % (g.ID, goroutine_status_to_string[g.Status])
        # frame_index counts the frames as presented by stack.Locations. For a
        # frame of interest, this index will later be used to eval() variables
        # in the right scope.
        frame_index = 0
        # output_frame_index is like frame_index, but doesn't get incremented
        # for frames that we don't include in the output. This will be used to
        # associate the data about a frame of interest with the output stack
        # frames.
        output_frame_index = 0
        for f in stack.Locations:
            if f.Location.Function:
                fun_name = f.Location.Function.Name_
            else:
                # TODO(andrei): if we don't have a function name, this is some
                # some assembly code towards the bottom of the stack. We skip
                # this frame because I'm not sure how to write something that
                # panicparse will accept.
                # fun_name = '<unknown>'
                frame_index = frame_index + 1
                continue
            backtrace = backtrace + '%s()\n\t%s:%d\n' % (fun_name, f.Location.File, f.Location.Line)
            for function_of_interest in frames_of_interest:
                if f.Location.Function.Name_.endswith(function_of_interest):
                    recognized_frames.append(struct(
                        gid=g.ID,
                        function_of_interest=function_of_interest,
                        frame_index=frame_index,
                        output_frame_index=output_frame_index,
                    ))
                    # print("backtrace for %d:\n%s" % (g.ID, serialize_backtrace(g.ID, output_frame_index)))

                    # res.append((g.ID, frame_index, function_of_interest, f.Location.Function.Name_))
            frame_index = frame_index + 1
            output_frame_index = output_frame_index + 1
        g_out[g.ID] = backtrace

    # Evaluate the expressions for all the frames of interest.
    # vars will be map of int (gid) to map of int (frame index) to list of
    # strings.
    vars = {}
    for var in recognized_frames:
        for expr in frames_of_interest[var.function_of_interest]:
            val = eval(
                {"GoroutineID": var.gid, "Frame": var.frame_index},
                expr,
                {"FollowPointers": True, "MaxVariableRecurse": 2, "MaxStringLen": 100,
                 "MaxArrayValues": 10, "MaxStructFields": 100}
            ).Variable.Value
            vars.setdefault(var.gid, {})
            vars[var.gid].setdefault(var.output_frame_index, [])
            vars[var.gid][var.output_frame_index].append(str(val))

    print("looked at #goroutines: ", len(gs))
    output = {
        "stacks": g_out,
        "frames_of_interest": vars,
    }
    return json.encode(output)


def main():
    return gs()
