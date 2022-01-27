package statemachine

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/looplab/fsm"
	"github.com/nicholasjackson/consul-release-controller/models"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	plugins "github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
)

// StepDelay is used to set the default delay between events
var StepDelay = 10 * time.Second

// defaultTimeout is the default time that an event step can take before timing out
var defaultTimeout = 30 * time.Minute

type StateMachine struct {
	release        *models.Release
	releaserPlugin interfaces.Releaser
	runtimePlugin  interfaces.Runtime
	monitorPlugin  interfaces.Monitor
	strategyPlugin interfaces.Strategy
	logger         hclog.Logger
	metrics        interfaces.Metrics

	metricsDone func(int)

	stateHistory []interfaces.StateHistory

	*fsm.FSM
}

func New(r *models.Release, pluginProvider interfaces.Provider) (*StateMachine, error) {
	sm := &StateMachine{release: r}
	sm.stateHistory = []interfaces.StateHistory{interfaces.StateHistory{Time: time.Now(), State: StateStart}}
	sm.logger = pluginProvider.GetLogger().Named("statemachine")
	sm.metrics = pluginProvider.GetMetrics()

	// configure the setup plugin
	relP, err := pluginProvider.CreateReleaser(r.Releaser.Name)
	if err != nil {
		return nil, err
	}

	// configure the releaser plugin
	relP.Configure(r.Releaser.Config)
	sm.releaserPlugin = relP

	// configure the runtime plugin
	runP, err := pluginProvider.CreateRuntime(r.Runtime.Name)
	if err != nil {
		return nil, err
	}

	// configure the runtime plugin
	runP.Configure(r.Runtime.Config)
	sm.runtimePlugin = runP

	// configure the monitor plugin
	monP, err := pluginProvider.CreateMonitor(r.Monitor.Name)
	if err != nil {
		return nil, err
	}

	// configure the monitor plugin
	rc := runP.BaseConfig()
	monP.Configure(rc.Deployment, rc.Namespace, r.Runtime.Name, r.Monitor.Config)
	sm.monitorPlugin = monP

	// configure the monitor plugin
	stratP, err := pluginProvider.CreateStrategy(r.Strategy.Name, monP)
	if err != nil {
		return nil, err
	}

	// configure the strategy plugin
	stratP.Configure(r.Name, r.Namespace, r.Strategy.Config)
	sm.strategyPlugin = stratP

	f := fsm.NewFSM(
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
			{Name: EventComplete, Src: []string{StateDestroy}, Dst: StateIdle},
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
			"before_event":            sm.logEvent(),
			"enter_" + StateConfigure: sm.doConfigure(), // do the necessary work to setup the release
			"enter_" + StateDeploy:    sm.doDeploy(),    // new version of the application has been deployed
			"enter_" + StateMonitor:   sm.doMonitor(),   // start monitoring changes in the applications health
			"enter_" + StateScale:     sm.doScale(),     // scale the release
			"enter_" + StatePromote:   sm.doPromote(),   // promote the release to primary
			"enter_" + StateRollback:  sm.doRollback(),  // rollback the deployment
			"enter_" + StateDestroy:   sm.doDestroy(),   // remove everything and revert to vanilla state
			"enter_state":             sm.enterState(),
			"leave_state":             sm.leaveState(),
		},
	)

	f.SetState(StateStart)
	sm.FSM = f

	return sm, nil
}

// Configure triggers the EventConfigure state
func (s *StateMachine) Configure() error {
	return s.Event(EventConfigure)
}

// Deploy triggers the EventDeploy state
func (s *StateMachine) Deploy() error {
	return s.Event(EventDeploy)
}

// Destroy triggers the event Destroy state
func (s *StateMachine) Destroy() error {
	return s.Event(EventDestroy)
}

// CurrentState returns the current state of the machine
func (s *StateMachine) CurrentState() string {
	return s.FSM.Current()
}

// CurrentState returns the current state of the machine
func (s *StateMachine) StateHistory() []interfaces.StateHistory {
	return s.stateHistory
}

func (s *StateMachine) logEvent() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Handle event", "event", e.Event, "state", e.FSM.Current())
	}
}

func (s *StateMachine) enterState() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Log state", "event", e.Event, "release", s.release.Name, "state", e.FSM.Current())

		// setup timing for the duration we exist in this state
		s.metricsDone = s.metrics.StateChanged(s.release.Name, e.FSM.Current(), nil)

		// append the state history
		s.stateHistory = append(s.stateHistory, interfaces.StateHistory{Time: time.Now(), State: e.FSM.Current()})
	}
}

func (s *StateMachine) leaveState() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Log state", "event", e.Event, "state", e.FSM.Current())

		// when we leave the state call the timing done function
		if s.metricsDone != nil {
			s.metricsDone(1)
		}
	}
}

func (s *StateMachine) doConfigure() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Configure", "state", e.FSM.Current())
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()

			// setup the initial configuration
			err := s.releaserPlugin.Setup(ctx)
			if err != nil {
				s.logger.Error("Configure completed with error", "error", err)

				e.FSM.Event(EventFail)
				return
			}

			// if a deployment already exists copy this to the primary
			status, err := s.runtimePlugin.InitPrimary(ctx)
			if err != nil {
				s.logger.Error("Configure completed with error", "status", status, "error", err)

				e.FSM.Event(EventFail)
				return
			}

			// if we created a new primary, scale all traffic to the new primary
			if status == interfaces.RuntimeDeploymentUpdate {
				err = s.releaserPlugin.Scale(ctx, 0)
				if err != nil {
					s.logger.Error("Configure completed with error", "error", err)

					e.FSM.Event(EventFail)
					return
				}

				// remove the candidate
				err = s.runtimePlugin.RemoveCandidate(ctx)
				if err != nil {
					s.logger.Error("Configure completed with error", "error", err)

					e.FSM.Event(EventFail)
					return
				}
			}

			s.logger.Debug("Configure completed successfully")

			e.FSM.Event(EventConfigured)
		}()
	}
}

func (s *StateMachine) doDeploy() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Deploy", "state", e.FSM.Current())
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

		go func() {
			// wait a few seconds as deploy is called before the new deployment is admitted to the server
			time.Sleep(StepDelay)

			// clean up resources if we finish before timeout
			defer cancel()

			// Create a primary if one does not exist
			status, err := s.runtimePlugin.InitPrimary(ctx)
			if err != nil {
				s.logger.Error("Deploy completed with error", "error", err)

				e.FSM.Event(EventFail)
				return
			}

			// now the primary has been created send 100 of traffic there
			err = s.releaserPlugin.Scale(ctx, 0)
			// work has failed, raise the failed event
			if err != nil {
				s.logger.Error("Deploy completed with error", "error", err)

				e.FSM.Event(EventFail)
				return
			}

			// if we created a primary this is the first deploy, no need to execute the strategy
			if status == interfaces.RuntimeDeploymentUpdate {
				s.logger.Debug("Deploy completed, created primary, waiting for next candidate deployment")

				// remove the candidate and wait for the next deployment
				err = s.runtimePlugin.RemoveCandidate(ctx)
				if err != nil {
					s.logger.Error("Deploy completed with error", "error", err)

					e.FSM.Event(EventFail)
					return
				}

				e.FSM.Event(EventComplete)
				return
			}

			// new deployment run the strategy
			s.logger.Debug("Deploy completed, executing strategy")
			e.FSM.Event(EventDeployed)
		}()
	}
}

func (s *StateMachine) doMonitor() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Monitor", "state", e.FSM.Current())
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()

			result, traffic, err := s.strategyPlugin.Execute(ctx)

			// strategy has failed with an error
			if err != nil {
				s.logger.Error("Monitor completed with error", "error", err)

				e.FSM.Event(EventFail)
			}

			// strategy returned a response
			switch result {
			// when the strategy reports a healthy deployment
			case plugins.StrategyStatusSuccess:
				// send the traffic with the healthy event so that it can be used for scaling
				s.logger.Debug("Monitor checks completed, candidate healthy")

				e.FSM.Event(EventHealthy, traffic)

			// the strategy has completed the roll out promote the deployment
			case plugins.StrategyStatusComplete:
				s.logger.Debug("Monitor checks completed, strategy complete")

				e.FSM.Event(EventComplete)

			// the strategy has reported that the deployment is unhealthy, rollback
			case plugins.StrategyStatusFail:
				s.logger.Debug("Monitor checks completed, candidate unhealthy")

				e.FSM.Event(EventUnhealthy)
			}
		}()
	}
}

func (s *StateMachine) doScale() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Scale", "state", e.FSM.Current())
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()

			// get the traffic from the event
			if len(e.Args) != 1 {
				s.logger.Error("Scale completed with error", "error", fmt.Errorf("no traffic percentage in event payload"))

				e.FSM.Event(EventFail)
				return
			}

			traffic := e.Args[0].(int)

			err := s.releaserPlugin.Scale(ctx, traffic)
			if err != nil {
				s.logger.Error("Scale completed with error", "error", err)

				e.FSM.Event(EventFail)
				return
			}

			s.logger.Debug("Scale completed successfully")
			e.FSM.Event(EventScaled)
		}()
	}
}

func (s *StateMachine) doPromote() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Promote", "state", e.FSM.Current())
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()

			// scale all traffic to the candidate before promoting
			err := s.releaserPlugin.Scale(ctx, 100)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// promote the candidate to primary
			_, err = s.runtimePlugin.PromoteCandidate(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// scale all traffic to the primary
			err = s.releaserPlugin.Scale(ctx, 0)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// scale down the canary
			err = s.runtimePlugin.RemoveCandidate(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			e.FSM.Event(EventPromoted)
		}()
	}
}

func (s *StateMachine) doRollback() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Rollback", "state", e.FSM.Current())
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			// scale all traffic to the primary
			err := s.releaserPlugin.Scale(ctx, 0)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// scale down the canary
			err = s.runtimePlugin.RemoveCandidate(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			e.FSM.Event(EventComplete)
		}()
	}
}

func (s *StateMachine) doDestroy() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		s.logger.Debug("Destroy", "state", e.FSM.Current())

		go func() {
			// clean up resources if we finish before timeout
			defer cancel()
			// restore the original deployment
			err := s.runtimePlugin.RestoreOriginal(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// scale all traffic to the candidate
			err = s.releaserPlugin.Scale(ctx, 100)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// destroy the primary
			err = s.runtimePlugin.RemovePrimary(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			// remove the consul config
			err = s.releaserPlugin.Destroy(ctx)
			if err != nil {
				e.FSM.Event(EventFail)
				return
			}

			e.FSM.Event(EventComplete)
		}()
	}
}
