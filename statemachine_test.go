package statemachine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type StateMachineSuite struct {
	suite.Suite
	SM            *StateMachine
	executionChan chan struct{}
}

func (suite *StateMachineSuite) SetupTest() {
	suite.SM = NewStateMachine(map[Key]*Transition{
		GetKey("idle", "go"): {
			To:    "running",
			Delta: func() error { return nil },
		},
		GetKey("running", "stop"): {
			To:    "idle",
			Delta: func() error { return nil },
		},
	}, "idle")

	suite.executionChan = make(chan struct{}, 128)
}

func (suite *StateMachineSuite) waitForExecution(number int) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	counter := 0
	for {
		select {
		case <-ctx.Done():
			suite.Failf("got statemachine timeout", "expected %d executions but got %d", number, counter)
			return
		case <-suite.executionChan:
			counter++
			if counter == number {
				return
			}
		}
	}
}

func (suite *StateMachineSuite) TestTransitions() {
	suite.Equal(State("idle"), suite.SM.Current(), "Initial state should be 'idle'")

	suite.SM.Trigger("go", func(err error, state State) {
		suite.NoError(err, "Transition idle -> (go) -> running should not produce an error")
		suite.Equal(State("running"), state, "State should be 'running' after 'idle'")
		suite.executionChan <- struct{}{}
	})

	suite.SM.Trigger("stop", func(err error, state State) {
		suite.NoError(err, "Transition running -> stop should not produce an error")
		suite.Equal(State("idle"), state, "State should be 'idle' after 'stop'")
		suite.executionChan <- struct{}{}
	})

	suite.waitForExecution(2)
}

func (suite *StateMachineSuite) TestInvalidTransition() {
	suite.SM.Trigger("pause", func(err error, state State) {
		suite.Error(err, "Triggering 'pause' should produce an error")
		suite.executionChan <- struct{}{}
	})

	suite.waitForExecution(1)
}

func (suite *StateMachineSuite) TestDuplicateEvents() {
	suite.SM.AddTransition("idle", "nop", &Transition{
		To:    "idle",
		Delta: func() error { return nil },
	})

	suite.Equal(State("idle"), suite.SM.Current(), "Initial state should be 'idle'")

	suite.SM.Trigger("nop", func(err error, state State) {
		suite.NoError(err, "Transition idle -> nop -> idle should not produce an error in idle state")
		suite.Equal(State("idle"), state, "State should be 'idle' after 'nop'")
		suite.executionChan <- struct{}{}
	})

	suite.SM.Trigger("go", func(err error, state State) {
		suite.NoError(err, "Transition idle -> go -> running should not produce an error")
		suite.Equal(State("running"), state, "State should be 'running' after 'start'")
		suite.executionChan <- struct{}{}
	})

	suite.SM.Trigger("stop", func(err error, state State) {
		suite.NoError(err, "Transition running -> stop -> idle should not produce an error")
		suite.Equal(State("idle"), state, "State should be 'idle' after 'stop'")
		suite.executionChan <- struct{}{}
	})

	suite.waitForExecution(3)
}

func (suite *StateMachineSuite) TestErrorAssertion() {
	suite.Equal(State("idle"), suite.SM.Current(), "Initial state should be 'idle'")

	suite.SM.Trigger("undefined", func(err error, state State) {
		suite.True(IsTransitionUndefinedError(err))
		suite.executionChan <- struct{}{}
	})

	suite.waitForExecution(1)
}

func (suite *StateMachineSuite) TestMassParallel() {
	suite.Equal(State("idle"), suite.SM.Current(), "Initial state should be 'idle'")

	for i := 0; i < 128; i++ {
		suite.SM.Trigger("undefined", func(err error, state State) {
			suite.True(IsTransitionUndefinedError(err))
			suite.executionChan <- struct{}{}
		})
	}

	suite.waitForExecution(128)
}

func (suite *StateMachineSuite) TestSequentialExecution() {
	suite.SM.AddTransition("idle", "nop", &Transition{
		To:    "idle",
		Delta: func() error { return nil },
	})

	suite.Equal(State("idle"), suite.SM.Current(), "Initial state should be 'idle'")

	err := <-suite.SM.Trigger("nop")
	suite.NoError(err.Error, "Transition idle -> nop -> idle should not produce an error in idle state")
	suite.Equal(State("idle"), suite.SM.Current(), "State should be 'idle' after 'nop'")

	err = <-suite.SM.Trigger("go")
	suite.NoError(err.Error, "Transition idle -> go -> running should not produce an error")
	suite.Equal(State("running"), suite.SM.Current(), "State should be 'running' after 'start'")

	err = <-suite.SM.Trigger("stop")
	suite.NoError(err.Error, "Transition running -> stop -> idle should not produce an error")
	suite.Equal(State("idle"), suite.SM.Current(), "State should be 'idle' after 'stop'")
}

func TestStateMachineSuite(t *testing.T) {
	suite.Run(t, new(StateMachineSuite))
}
