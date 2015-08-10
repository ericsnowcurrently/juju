// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package workers_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/process"
	"github.com/juju/juju/process/workers"
	"github.com/juju/juju/testing"
)

type eventHandlerSuite struct {
	testing.BaseSuite
}

var _ = gc.Suite(&eventHandlerSuite{})

func (s *eventHandlerSuite) TestNewEventHandlers(c *gc.C) {
	eh := workers.NewEventHandlers()
	defer eh.Close()

	// TODO(ericsnow) This test is rather weak.
	c.Check(eh, gc.NotNil)
}

func (s *eventHandlerSuite) TestRegisterHandler(c *gc.C) {
	eh := workers.NewEventHandlers()
	defer eh.Close()
	handler := func([]process.Event) error { return nil }
	eh.RegisterHandler(handler)

	// TODO(ericsnow) Check something here.
}

func (s *eventHandlerSuite) TestAddEvents(c *gc.C) {
	events := []process.Event{{
		Kind: process.EventKindTracked,
		ID:   "spam/eggs",
	}}
	//eventsCh := make(chan []process.Event, 2)
	eh := workers.NewEventHandlers()
	go func() {
		eh.AddEvents(events...)
		eh.Close()
	}()

	var got [][]process.Event
	for event := range workers.ExposeChannel(eh) {
		got = append(got, event)
	}
	c.Check(got, jc.DeepEquals, [][]process.Event{events})
}

func (s *eventHandlerSuite) TestNewWorker(c *gc.C) {
	events := []process.Event{{
		Kind: process.EventKindTracked,
		ID:   "spam/eggs",
	}}
	eh := workers.NewEventHandlers()
	var handled [][]process.Event
	handler := func(events []process.Event) error {
		handled = append(handled, events)
		return nil
	}
	eh.RegisterHandler(handler)
	w, err := eh.NewWorker()
	c.Assert(err, jc.ErrorIsNil)

	eh.AddEvents(events...)

	w.Kill()
	err = w.Wait()
	c.Assert(err, jc.ErrorIsNil)
	eh.Close()

	var unhandled [][]process.Event
	for event := range workers.ExposeChannel(eh) {
		unhandled = append(unhandled, event)
	}
	c.Check(unhandled, gc.HasLen, 0)
	c.Check(handled, jc.DeepEquals, [][]process.Event{events})
}
