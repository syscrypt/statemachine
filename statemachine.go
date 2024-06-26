package statemachine

import (
	"fmt"
	"regexp"
	"sync"
)

// State and Event names should comply with the following
// regular expression: [a-zA-Z_]+[a-zA-Z0-9_]*
type State string
type Event string
type Key string

var (
	isUndefTransErrRegex = regexp.MustCompile(
		"transition from state [a-zA-Z_]+[a-zA-Z0-9_]* with event [a-zA-Z_]+[a-zA-Z0-9_]* does not exist",
	)
)

type StateMachine struct {
	mut       sync.Mutex
	states    map[Key]*Transition
	current   State
	eventChan chan *triggerOp
}

type Transition struct {
	To    State
	Delta func() error
}

type StateMachineResult struct {
	Error error
	State State
}

type triggerOp struct {
	event           Event
	resultProcessor []func(err error, state State)
	resultChan      chan *StateMachineResult
}

func NewStateMachine(deltas map[Key]*Transition, initial State) *StateMachine {
	m := &StateMachine{
		mut:       sync.Mutex{},
		states:    make(map[Key]*Transition),
		current:   initial,
		eventChan: make(chan *triggerOp, 1024),
	}
	m.AddTransitions(deltas)
	return m
}

func (m *StateMachine) AddTransition(from State, event Event, transition *Transition) {
	m.states[GetKey(from, event)] = transition
}

func (m *StateMachine) AddTransitions(deltas map[Key]*Transition) {
	for key, f := range deltas {
		m.states[key] = f
	}
}

func (m *StateMachine) Current() State {
	return m.current
}

func (m *StateMachine) Trigger(event Event, resultProcessor ...func(err error, state State)) <-chan *StateMachineResult {
	defer func() {
		go m.trigger()
	}()

	resultChan := make(chan *StateMachineResult, 2)
	m.eventChan <- &triggerOp{
		event:           event,
		resultProcessor: resultProcessor,
		resultChan:      resultChan,
	}
	return resultChan
}

func IsTransitionUndefinedError(err error) bool {
	if err == nil {
		return false
	}
	return isUndefTransErrRegex.MatchString(err.Error())
}

func GetKey(from State, event Event) Key {
	return Key(fmt.Sprintf("%s->%s", from, event))
}

func (m *StateMachine) trigger() {
	var err error
	var transition *Transition

	m.mut.Lock()

	event := <-m.eventChan
	current := m.current

	defer func() {
		m.leaveTransition(err, transition, event.resultChan, event.resultProcessor...)
	}()

	key := GetKey(current, event.event)
	transition, ok := m.states[key]
	if !ok {
		err = fmt.Errorf("transition from state %s with event %s does not exist", current, event.event)
		return
	}

	if transition.Delta != nil {
		err = transition.Delta()
	}
}

func (m *StateMachine) leaveTransition(err error, transition *Transition,
	resultChan chan *StateMachineResult, resultProcessor ...func(err error, state State)) {
	if err == nil && transition != nil {
		m.current = transition.To
	}
	current := m.current

	m.mut.Unlock()

	resultChan <- &StateMachineResult{
		Error: err,
		State: current,
	}
	close(resultChan)

	for _, f := range resultProcessor {
		f(err, current)
	}
}
