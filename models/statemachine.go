package models

import (
	"context"
	"time"

	"github.com/looplab/fsm"
	"github.com/nicholasjackson/consul-canary-controller/plugins"
)

const (
	EventConfigure  = "event_configure"  // triggers the configuration of a new release
	EventConfigured = "event_configured" // fired when the release has been successfully configured
	EventDeploy     = "event_deploy"     // triggers a new deployment
	EventDeployed   = "event_deployed"   // fired when a new deployment has completed successfully
	EventHealthy    = "event_healthy"    // fired when a new deployment is healthy based on configured metrics
	EventUnhealthy  = "event_unhealthy"  // fired when a new deployment is unhealthy based on configured metrics
	EventScaled     = "event_scaled"     // fired when the release traffic has been scaled
	EventPromoted   = "event_promoted"   // fired when the new deployment has been promoted to active deployment
	EventComplete   = "event_complete"   // fired when all release traffic points at the new deployment
	EventFail       = "event_fail"       // fired when any state returns an error
	EventDestroy    = "event_destroy"    // triggers the destruction of a release

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

func newFSM(d *Release, s plugins.Releaser, r plugins.Runtime) *fsm.FSM {
	return fsm.NewFSM(
		StateStart,
		fsm.Events{
			{Name: EventConfigure, Src: []string{StateStart}, Dst: StateConfigure},
			{Name: EventConfigured, Src: []string{StateConfigure}, Dst: StateIdle},
			{Name: EventDeploy, Src: []string{StateIdle}, Dst: StateDeploy},
			{Name: EventDeployed, Src: []string{StateDeploy}, Dst: StateMonitor},
			{Name: EventHealthy, Src: []string{StateMonitor}, Dst: StateScale},
			{Name: EventScaled, Src: []string{StateScale}, Dst: StateMonitor},
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
			"enter_" + StateConfigure: doAsync(s.Setup), // do the necessary work
			"enter_" + StateDeploy:    doAsync(r.Deploy),
			"enter_" + StateIdle:      saveRelease(d),
		},
	)
}

var defaultTimeout = 30 * time.Minute

func saveRelease(r *Release) func(e *fsm.Event) {
	return func(e *fsm.Event) {
		r.Save(e.Dst)
	}
}

// wrapTimeout ensures that the state function is executed asynchronously
func doAsync(f func(ctx context.Context) error) func(e *fsm.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	return func(e *fsm.Event) {
		go func() {
			// clean up resources if we finish before timeout
			defer cancel()

			// execute the work function
			err := f(ctx)

			// work has failed, raise the failed event
			if err != nil {
				e.Cancel(err)
				e.FSM.SetState(StateFail)
			}

			// work has succeeded notify the callback
			e.Args[0].(func(*fsm.Event))(e)
		}()
	}
}
