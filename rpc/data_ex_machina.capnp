using Go = import "/go.capnp";
@0xa5f26ade549e9db5;
$Go.package("rpc");
$Go.import("github.com/andreimatei/delve-agent/rpc");


interface Stream(T) {
    # Returns the next element in the stream.
    #
    # Does not return until the next element is available.
    getNext @0 () -> (next :T);
}

interface Lifecycle {
    waitEnd @0 () -> ();
}

struct Config {
    struct Predicate {
        # Regex to match against the executable path.
        #
        # Considered to match if the regex matches any part of the path.
        #
        # Use ^ and $ to match the entire path.
        binaryPathRegex @0 :Text;
    }
    predicates @0 :List(Predicate);
}

# TODO: this should just be a type alias, but that feature is not implemented. See
# https://github.com/capnproto/capnproto/issues/907.
struct Identifier {
    # TODO: this should be a fixed-width array, but that feature is not implemented. See
    # https://github.com/capnproto/capnproto/issues/205.
    identifier @0 :UInt64;
}

struct Process {
    interface Handle extends(Lifecycle) {}

    handle @0 :Handle;
    proc :group {
        pid @1 :UInt32;
        # Path to the executable.
        exe @2 :Data;
        # NUL-separated list of arguments.
        cmdline @3 :Data;
    }
    # Content-addressable identifier for the executable.
    executable @4 :Identifier;
}

interface Ex {
    # Returns a stream of newly discovered machinas. The stream may yield machinas whose underlying
    # transport has failed.
    #
    # Implementations should prune the stream of machinas that are no longer reachable.
    #
    # Note that machinas are typically remote machines; transient failures may lead to "the same"
    # machina appearing on the stream multiple times.
    #
    # Calls on the machina may fail if the underlying connection to the machina fails.
    getMachinas @0 () -> (stream :Stream(List(Machina)));

    struct Machina {
        # TODO: extend Lifecycle so clients can learn of a Machina's demise.
        interface Handle {
            # Streams newly discovered processes.
            #
            # Process exit can be detected using the Lifecycle interface on the Process handle.
            getProcesses @0 (config :Config) -> (stream :Stream(List(Process)));
        }

        handle @0 :Handle;
        version @1 :Text;
        # TODO: should this be more opaque?
        address @2 :Text;
    }

    getExecutable @1 (executable :Identifier) -> (executable :Executable);

    struct Executable {
        interface Handle {
            getDebug @0 () -> (debug :Debug);
        }
        handle @0 :Handle;
    }

    struct Debug {
        interface Handle {
            # TODO: pagination? maybe we don't need it. do we need dynamic limits?
            struct Query {
                # Whether to include the handle in the results.
                includeHandle @0 :Bool;
                # The depth to which recursive types are loaded.
                preloadDepth @1 :UInt32;
                # Must be contained in the name of the result.
                filter @2 :Text;
                # Maximum number of results to return. 0 means no limit.
                limit @3 :UInt32;
            }
            struct Function {
                struct Detail {
                    struct Variable {
                        struct Range {
                            start @0 :UInt64;
                            end @1 :UInt64;
                        }
                        name @0 :Text;
                        type @1 :Type;
                        # The range of program counters for which this variable can be read in its
                        # entirety.
                        pcs @2 :List(Range);
                    }
                    formalParameters @0 :List(Variable);
                    variables @1 :List(Variable);
                }
                interface Handle {
                    getDetail @0 () -> (detail :Detail);
                }
                handle @0 :Handle;
                name @1 :Text;
                detail @2 :Detail;
            }
            getFunctions @0 (query :Query) -> (functions :List(Function));
            getFunction @1 (name :Text) -> (handle :Function.Handle);
            struct Type {
                struct Detail {
                    struct Field {
                        name @0 :Text;
                        type @1 :Type;
                        union {
                            go :group {
                                embedded @2 :Bool;
                            }
                            rust :group {
                                placeholder @3 :UInt32;
                            }
                        }
                    }
                    fields @0 :List(Field);
                }
                interface Handle {
                    getDetail @0 () -> (detail :Detail);
                }
                handle @0 :Handle;
                name @1 :Text;
                detail @2 :Detail;
            }
            getTypes @2 (query :Query) -> (types :List(Type));
            getType @3 (name :Text) -> (handle :Type.Handle);
        }
        handle @0 :Handle;
    }
}

interface ExRegistrar {
    # Registers a machina.
    #
    # The registrar will use the machina to communicate with the registrant.
    #
    # The server never sends a response to this call. The implementation may use the outstanding
    # call to respond to transport failures.
    registerMachina @0 (machina :Machina) -> ();
}

interface ExMonolith extends(Ex, ExRegistrar) {}

interface Machina extends(Ex.Machina.Handle) {
    struct Executable {
        interface FileSink {
            sendChunk @0 (chunk :Data) -> stream;
            done @1 () -> ();
        }
        interface Handle {
            # TODO: compression?
            getContent @0 (sink :FileSink) -> ();
        }
        handle @0 :Handle;
    }
    getExecutable @0 (executable :Identifier) -> (executable :Executable);
}
