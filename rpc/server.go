// Copyright 2023 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package rpc

import (
	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"context"
	"log"
	"net"
	"time"
)

func ServeCapnproto(lis net.Listener) {
	ex := &ExMonolithServer{}
	exClient := capnp.Client(Ex_ServerToClient(ex))

	go func() {
		err := rpc.Serve(lis, exClient)
		log.Printf("capnp serving ended: %s", err)
	}()
}

// Maybe I only need Ex, not the "monolith" part with the machina registration.
type ExMonolithServer struct{}

var _ ExMonolith_Server = &ExMonolithServer{}

func (e *ExMonolithServer) GetMachinas(ctx context.Context, machinas Ex_getMachinas) error {
	res, err := machinas.AllocResults()
	if err != nil {
		return err
	}
	s := &StreamListMachinas{}
	c := Stream_ServerToClient(s)
	return res.SetStream(c)
}

func (e *ExMonolithServer) GetExecutable(ctx context.Context, executable Ex_getExecutable) error {
	//TODO implement me
	panic("implement me")
}

func (e *ExMonolithServer) RegisterMachina(ctx context.Context, machina ExRegistrar_registerMachina) error {
	//TODO implement me
	panic("implement me")
}

type StreamListMachinas struct {
	consumed bool
}

var _ Stream_Server = &StreamListMachinas{}

func (s *StreamListMachinas) GetNext(ctx context.Context, next Stream_getNext) error {
	res, err := next.AllocResults()
	if err != nil {
		return err
	}

	if s.consumed {
		// Block forever.
		<-ctx.Done()
		return ctx.Err()
	}
	s.consumed = true

	machina := &MachinaServer{}

	list, err := NewMachina_List(res.Segment(), 1)
	if err != nil {
		return err
	}
	err = list.Set(0, Machina_ServerToClient(machina))
	if err != nil {
		return err
	}
	res.SetNext(list.ToPtr())
	return nil
}

type MachinaServer struct{}

var _ Machina_Server = &MachinaServer{}

func (m MachinaServer) GetProcesses(ctx context.Context, processes Ex_Machina_Handle_getProcesses) error {
	res, err := processes.AllocResults()
	if err != nil {
		return err
	}
	s := &StreamListProcesses{}
	c := Stream_ServerToClient(s)
	return res.SetStream(c)
}

func (m MachinaServer) GetExecutable(ctx context.Context, executable Machina_getExecutable) error {
	//TODO implement me
	panic("implement me")
}

type StreamListProcesses struct {
	consumed bool
}

var _ Stream_Server = &StreamListProcesses{}

func (s *StreamListProcesses) GetNext(ctx context.Context, next Stream_getNext) error {
	log.Printf("!!! StreamListProcesses.GetNext()")
	res, err := next.AllocResults()
	if err != nil {
		return err
	}

	if s.consumed {
		// Block forever.
		<-ctx.Done()
		return ctx.Err()
	}
	s.consumed = true

	list, err := NewProcess_List(res.Segment(), 1)
	if err != nil {
		return err
	}
	proc, err := NewProcess(res.Segment())
	if err != nil {
		return err
	}
	proc.Proc().SetPid(4242)
	proc.SetHandle(Process_Handle_ServerToClient(&process{}))
	list.Set(0, proc)

	res.SetNext(list.ToPtr())
	return nil
}

type process struct{}

var _ Process_Handle_Server = &process{}

func (p *process) WaitEnd(ctx context.Context, end Lifecycle_waitEnd) error {
	time.Sleep(10 * time.Second)
	return nil
}
