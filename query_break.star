stmt = eval(None, "parserStmt.SQL")
flight_recorder(str(cur_scope().GoroutineID), stmt.Variable.Value)
