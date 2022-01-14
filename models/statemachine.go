package models

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/looplab/fsm"
	"github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
	plugins "github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
)

const (
	EventDeploy     = "event_deploy"     // triggers a new deployment
	EventDeployed   = "event_deployed"   // fired when a new deployment has completed successfully
	EventConfigure  = "event_configure"  // triggers the configuration of a new release
	EventConfigured = "event_configured" // fired when the release has been successfully configured
	EventHealthy    = "event_healthy"    // fired when a new deployment is healthy based on configured metrics
	EventUnhealthy  = "event_unhealthy"  // fired when a new deployment is unhealthy based on configured metrics
	EventScaled     = "event_scaled"     // fired when the release traffic has been scaled
	EventPromoted   = "event_promoted"   // fired when the new deployment has been promoted to active deployment
	EventComplete   = "event_complete"   // fired when all release traffic points at the new deployment
	EventFail       = "event_fail"       // fired when any state returns an error
	EventDestroy    = "event_destroy"    // triggers the destruction of a release
	EventNull       = "event_null"       // null event

	StateStart     = "state_start"     // initial state for a new release
	StateConfigure = "state_configure" // state when the release is currently configuring
	StateIdle      = "state_idle"      // state when the release is configured but inactive
	StateDeploy    = "state_deploy"    // state when the a new deployment is being created
	StateMonitor   = "state_monitor"   // state when the new deployment is being monitored for correctness
	StateScale     = "state_scale"     // state when the new deployment traffic is being scaled
	StatePromote   = "state_promote"   // state when the latest deployment is being promoted to active deployment
	StateRollback  = "state_rollback"  // state when the latest deployment is being removed
	StateFail      = "state_fail"      // state when the latest operation has failed
	StateDestroy   = "state_destroy"   // state when the release is being destroyed
)

func newFSM(r *Release, pRel plugins.Releaser, pRun plugins.Runtime, pStrat plugins.Strategy, l hclog.Logger) *fsm.FSM {
	return fsm.NewFSM(
		StateStart,
		fsm.Events{
			{Name: EventConfigure, Src: []string{StateStart, StateIdle, StateFail}, Dst: StateConfigure},
			{Name: EventConfigured, Src: []string{StateConfigure}, Dst: StateIdle},
			{Name: EventDeploy, Src: []string{StateIdle, StateFail}, Dst: StateDeploy},
			{Name: EventDeployed, Src: []string{StateDeploy}, Dst: StateMonitor},
			{Name: EventHealthy, Src: []string{StateMonitor}, Dst: StateScale},
			{Name: EventScaled, Src: []string{StateScale}, Dst: StateMonitor},
			{Name: EventComplete, Src: []string{StateMonitor}, Dst: StatePromote},
			{Name: EventPromoted, Src: []string{StatePromote}, Dst: StateIdle},
			{Name: EventUnhealthy, Src: []string{StateMonitor}, Dst: StateRollback},
			{Name: EventComplete, Src: []string{StateDeploy}, Dst: StateIdle},
			{Name: EventComplete, Src: []string{StateRollback}, Dst: StateIdle},
			{Name: EventFail, Src: []string{
				StateStart,
				StateConfigure,
				StateIdle,
				StateDeploy,
				StateMonitor,
				StateScale,
				StatePromote,
				StateRollback,
				StateDestroy,
			}, Dst: StateFail},
			{Name: EventDestroy, Src: []string{
				StateIdle,
				StateDeploy,
				StateMonitor,
				StateScale,
				StatePromote,
				StateRollback,
			}, Dst: StateDestroy},
		},
		fsm.Callbacks{
			"before_event":            logEvent(l),
			"enter_" + StateConfigure: doConfigure(pRel, pRun, r, l), // do the necessary work to setup the release
			"enter_" + StateDeploy:    doDeploy(pRun, pRel, r, l),    // new version of the application has been deployed
			"enter_" + StateMonitor:   doMonitor(pStrat, r, l),       // start monitoring changes in the applications health
			"enter_" + StateScale:     doScale(pRel, r, l),           // scale the release
			"enter_" + StatePromote:   doPromote(pRun, pRel, r, l),   // promote the release to primary
			"enter_" + StateRollback:  doRollback(pRun, pRel, r, l),  // rollback the deployment
			"enter_" + StateIdle:      saveRelease(r, l),             // everything is setup, wait for a deployment
			"enter_" + StateFail:      saveRelease(r, l),
			"enter_" + StateDestroy:   doDestroy(pRun, pRel, r, l), // remove everything and revert to vanilla state
			"enter_state":             logState(l, r),
			"leave_state":             logState(l, r),
		},
	)
}

var defaultTimeout = 30 * time.Minute

func logEvent(l hclog.Logger) func(e *fsm.Event) {
	return func(e *fsm.Event) {
		l.Debug("Handle event", "event", e.Event, "state", e.FSM.Current())
	}
}

func logState(l hclog.Logger, rel *Release) func(e *fsm.Event) {
	return func(e *fsm.Event) {
		l.Debug("Log state", "event", e.Event, "state", e.FSM.Current())
	}
}

func saveRelease(r *Release, l hclog.Logger) func(e *fsm.Event) {
	return func(e *fsm.Event) {
		l.Debug("Save release", "state", e.FSM.Current())

		r.Save(e.FSM.Current())
	}
}

func doConfigure(pRel plugins.Releaser, pRun plugins.Runtime, r *Release, l hclog.Logger) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		l.Debug("Configure", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			r.Save(e.FSM.Current())

			// execute the work function
			err := pRel.Setup(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// if a deployment already exists copy this to the primary
			status, err := pRun.InitPrimary(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// if we created a new primary, scale all traffic to the new primary
			if status == interfaces.RuntimeDeploymentUpdate || status == interfaces.RuntimeDeploymentNoAction {
				err = pRel.Scale(ctx, 0)
				if err != nil {
					e.FSM.Event(EventFail)
					return
				}

				// remove the canary
				err = pRun.RemoveCandidate(ctx)
				if err != nil {
					e.FSM.Event(EventFail)
					return
				}
			}

			e.FSM.Event(EventConfigured)
		}()
	}
}

func doDeploy(pRun plugins.Runtime, pRel plugins.Releaser, r *Release, l hclog.Logger) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		l.Debug("Deploy", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			r.Save(e.FSM.Current())

			// Create a primary if one does not exist
			status, err := pRun.InitPrimary(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// now the primary has been created send 100 of traffic there
			err = pRel.Scale(ctx, 0)
			// work has failed, raise the failed event
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// if we created a primary this is the first deploy, no need to canary
			if status == interfaces.RuntimeDeploymentUpdate {
				e.FSM.Event(EventComplete)
				return
			}

			// new deployment run the strategy
			e.FSM.Event(EventDeployed)
		}()
	}
}

func doMonitor(strat plugins.Strategy, r *Release, l hclog.Logger) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		l.Debug("Monitor", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			r.Save(e.FSM.Current())

			result, traffic, err := strat.Execute(ctx)

			// strategy has failed with an error
			if err != nil {
				l.Error("Monitor state failed", "error", err)

				e.FSM.Event(EventFail)
			}

			// strategy returned a response
			switch result {
			// when the strategy reports a healthy deployment
			case plugins.StrategyStatusSuccess:
				// send the traffic with the healthy event so that it can be used for scaling
				e.FSM.Event(EventHealthy, traffic)

			// the strategy has completed the roll out promote the deployment
			case plugins.StrategyStatusComplete:
				r.Save(e.FSM.Current())
				e.FSM.Event(EventComplete)

			// the strategy has reported that the deployment is unhealthy, rollback
			case plugins.StrategyStatusFail:
				e.FSM.Event(EventUnhealthy)
			}

		}()
	}
}

func doScale(rel plugins.Releaser, r *Release, l hclog.Logger) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		l.Debug("Scale", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			// save the state
			r.Save(e.FSM.Current())

			// get the traffic from the event
			if len(e.Args) != 1 {
				l.Error("Scale state failed", "error", fmt.Errorf("no traffic percentage in event payload"))

				e.FSM.Event(EventFail)
				return
			}

			traffic := e.Args[0].(int)

			err := rel.Scale(ctx, traffic)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			e.FSM.Event(EventScaled)
		}()
	}
}

func doPromote(run plugins.Runtime, rel plugins.Releaser, r *Release, l hclog.Logger) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		l.Debug("Promote", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			// save the state
			r.Save(e.FSM.Current())

			// scale all traffic to the canary before promoting
			err := rel.Scale(ctx, 100)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// promote the canary to primary
			_, err = run.PromoteCandidate(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// scale all traffic to the primary
			err = rel.Scale(ctx, 0)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// scale down the canary
			err = run.RemoveCandidate(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			e.FSM.Event(EventPromoted)
		}()
	}
}

func doRollback(run plugins.Runtime, rel plugins.Releaser, r *Release, l hclog.Logger) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		l.Debug("Rollback", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			// save the state
			r.Save(e.FSM.Current())

			// scale all traffic to the primary
			err := rel.Scale(ctx, 0)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// scale down the canary
			err = run.RemoveCandidate(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			e.FSM.Event(EventComplete)
		}()
	}
}

func doDestroy(run plugins.Runtime, rel plugins.Releaser, r *Release, l hclog.Logger) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		l.Debug("Destroy", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			// save the state
			r.Save(e.FSM.Current())

			// restore the original deployment
			err := run.RestoreOriginal(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// scale all traffic to the canary
			err = rel.Scale(ctx, 100)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// destroy the primary
			err = run.RemovePrimary(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// remove the consul config
			err = rel.Destroy(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			e.FSM.Event(EventComplete)
		}()
	}
}
