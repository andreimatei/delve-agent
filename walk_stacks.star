# Maps from function name to list of expressions to evaluate in the scope of
# that function.
# frames_of_interest = {
#     'execStmtInOpenState': ['parserStmt.SQL', 'p.semaCtx.Placeholders.Values'],
#     # 'executeWriteBatch': 'ba',
#     # 'executeRead': 'ba',
#     # 'executeRead': 'ba.Requests[0].Value.(*kvpb.RequestUnion_Get).Get'
# }
frames_of_interest = {
    $frames_spec
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

# def getSpanNameFromCtx(gid, frame_idx, lastCtx):
#     val = eval(
#         {"GoroutineID": gid, "Frame": frame_idx},
#         "ctx",
#         {"FollowPointers": False}
#     )
#     if val != None:
#         print(val.Variable)

def gs():
    gs = goroutines().Goroutines

    # recognized_frames accumulates info about frames for which we'll evaluate
    # some expressions.
    recognized_frames = []
    g_out = {}
    # vars will be map of int (gid) to map of int (frame index) to list of
    # strings.
    vars = {}
    for g in gs:
        print("======= GOROUTINE ", g.ID)
        stack = stacktrace(g.ID,
                           200,  # depth
                           False,  # full
                           False,  # defers
                           # 7,     # option flags
                           # {"FollowPointers":True, "MaxVariableRecurse":3, "MaxStringLen":0, "MaxArrayValues":10, "MaxStructFields":100}, # MaxVariableRecurse:1, MaxStringLen:64, MaxArrayValues:64, MaxStructFields:-1}"
                           ContextExprs=True,
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
            backtrace = backtrace + '%s()\n\t%s:%d +0x%x\n' % (
                fun_name, f.Location.File, f.Location.Line, f.Location.PC - f.Location.Function.EntryPC)
            op = ""
            if len(f.CtxExpressions) > 0:
                op = f.CtxExpressions[0].Value
            print(g.ID, fun_name, op, len(f.CtxExpressions))
            for function_of_interest in frames_of_interest:
                if f.Location.Function.Name_.endswith(function_of_interest):
                    recognized_frames.append(struct(
                        gid=g.ID,
                        function_of_interest=function_of_interest,
                        frame_index=frame_index,
                        output_frame_index=output_frame_index,
                    ))

            if len(f.CtxExpressions) > 0:
                op = f.CtxExpressions[0].Value
                vars.setdefault(g.ID, {})
                vars[g.ID][output_frame_index] = [{"Expr": "span.op", "Val": op}]

            frame_index = frame_index + 1
            output_frame_index = output_frame_index + 1
        g_out[g.ID] = backtrace

    # Evaluate the expressions for all the frames of interest.
    for frame in recognized_frames:
        for expr in frames_of_interest[frame.function_of_interest]:
            val = eval(
                {"GoroutineID": frame.gid, "Frame": frame.frame_index},
                expr,
                {"FollowPointers": True, "MaxVariableRecurse": 2, "MaxStringLen": 100,
                 "MaxArrayValues": 10, "MaxStructFields": 100}
            ).Variable.Value
            vars.setdefault(frame.gid, {})
            vars[frame.gid].setdefault(frame.output_frame_index, [])
            vars[frame.gid][frame.output_frame_index].append({"Expr": expr, "Val": str(val)})

    print("looked at #goroutines: ", len(gs))
    output = {
        "stacks": g_out,
        "frames_of_interest": vars,
    }
    return json.encode(output)


def main():
    return gs()
