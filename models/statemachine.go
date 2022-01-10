package models

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/looplab/fsm"
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

/*
inactive
	-> *initalize -> initializing

initializing
	-> *fail -> failed
	-> *initialized -> initialized

initialized
	-> *cancel -> canceling
*/

func newFSM(r *Release, pRel plugins.Releaser, pRun plugins.Runtime, pStrat plugins.Strategy, l hclog.Logger) *fsm.FSM {
	return fsm.NewFSM(
		StateStart,
		fsm.Events{
			{Name: EventDeploy, Src: []string{StateStart}, Dst: StateDeploy},
			{Name: EventDeployed, Src: []string{StateDeploy}, Dst: StateConfigure},
			{Name: EventConfigured, Src: []string{StateConfigure}, Dst: StateScale},
			{Name: EventScaled, Src: []string{StateScale}, Dst: StateMonitor},
			{Name: EventHealthy, Src: []string{StateConfigure, StateMonitor}, Dst: StateScale},
			{Name: EventComplete, Src: []string{StateMonitor}, Dst: StatePromote},
			{Name: EventPromoted, Src: []string{StatePromote}, Dst: StateIdle},
			{Name: EventUnhealthy, Src: []string{StateMonitor}, Dst: StateRollback},
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
			"enter_" + StateDeploy:    doDeploy(pRun, r, l),          // new version of the application has been deployed
			"enter_" + StateConfigure: doConfigure(pRel.Setup, r, l), // do the necessary work to setup the release
			"enter_" + StateMonitor:   doMonitor(pStrat, r, l),       // start monitoring changes in the applications health
			"enter_" + StateScale:     doScale(pRel, r, l),           // scale the release
			"enter_" + StatePromote:   doPromote(pRun, pRel, r, l),   // promote the release to primary
			"enter_" + StateRollback:  saveRelease(r, l),             // rollback the deployment
			"enter_" + StateIdle:      saveRelease(r, l),             // everything is setup, wait for a deployment
			"enter_" + StateFail:      saveRelease(r, l),
		},
	)
}

var defaultTimeout = 30 * time.Minute

func logEvent(l hclog.Logger) func(e *fsm.Event) {
	return func(e *fsm.Event) {
		l.Debug("handle event", "event", e.Event, "state", e.FSM.Current())
	}
}

func saveRelease(r *Release, l hclog.Logger) func(e *fsm.Event) {
	return func(e *fsm.Event) {
		l.Debug("save release", "state", e.FSM.Current())

		r.Save(e.Dst)
	}
}

func doDeploy(pRun plugins.Runtime, r *Release, l hclog.Logger) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		l.Debug("deploy", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()

			r.Save(e.FSM.Current())

			// execute the work function
			err := pRun.Deploy(ctx)

			// work has failed, raise the failed event
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			e.FSM.Event(EventDeployed)
		}()
	}
}

func doConfigure(f func(ctx context.Context) error, r *Release, l hclog.Logger) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		l.Debug("configure", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()

			r.Save(e.FSM.Current())

			// execute the work function
			err := f(ctx)

			// work has failed, raise the failed event
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// call the EventConfigured setting the traffic to -1 so that initial traffic is set
			e.FSM.Event(EventConfigured, -1)
		}()
	}
}

func doMonitor(strat plugins.Strategy, r *Release, l hclog.Logger) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		l.Debug("monitor", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()

			r.Save(e.FSM.Current())
			result, traffic, err := strat.Execute(ctx)

			// strategy has failed with an error
			if err != nil {
				l.Error("monitor state failed", "error", err)

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
		l.Debug("scale", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			// save the state
			r.Save(e.FSM.Current())

			// get the traffic from the event
			if len(e.Args) != 1 {
				l.Error("scale state failed", "error", fmt.Errorf("no traffic percentage in event payload"))
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
		l.Debug("promote", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			// save the state
			r.Save(e.FSM.Current())

			err := run.Promote(ctx)
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

			e.FSM.Event(EventPromoted)
		}()
	}
}
