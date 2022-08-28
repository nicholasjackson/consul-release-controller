package statemachine

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/looplab/fsm"
	"github.com/nicholasjackson/consul-release-controller/pkg/models"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
)

// stepDelay is used to set the default delay between events
var stepDelay = 5 * time.Second

// defaultTimeout is the default time that an event step can take before timing out
var defaultTimeout = 30 * time.Minute

type StateMachine struct {
	release        *models.Release
	releaserPlugin interfaces.Releaser
	runtimePlugin  interfaces.Runtime
	monitorPlugin  interfaces.Monitor
	strategyPlugin interfaces.Strategy
	testPlugin     interfaces.PostDeploymentTest
	webhookPlugins []interfaces.Webhook
	logger         hclog.Logger
	metrics        interfaces.Metrics
	storage        interfaces.Store

	metricsDone func(int)

	*fsm.FSM
}

func New(r *models.Release, pluginProvider interfaces.Provider) (*StateMachine, error) {
	sm := &StateMachine{release: r, webhookPlugins: []interfaces.Webhook{}}
	sm.logger = pluginProvider.GetLogger().Named("statemachine")
	sm.metrics = pluginProvider.GetMetrics()
	sm.storage = pluginProvider.GetDataStore()

	// create the setup plugin
	relP, err := pluginProvider.CreateReleaser(r.Releaser.Name)
	if err != nil {
		return nil, err
	}

	// configure the releaser plugin
	relP.Configure(r.Releaser.Config, sm.logger.ResetNamed("releaser-plugin"), sm.storage.CreatePluginStateStore(r, "releaser"))
	sm.releaserPlugin = relP

	// get the releaser config
	releaserConfig := relP.BaseConfig()

	// configure the runtime plugin
	runP, err := pluginProvider.CreateRuntime(r.Runtime.Name)
	if err != nil {
		return nil, err
	}

	//p.kubeClient = kc

	// configure the runtime plugin
	runP.Configure(r.Runtime.Config, sm.logger.ResetNamed("runtime-plugin"), sm.storage.CreatePluginStateStore(r, "runtime"))
	sm.runtimePlugin = runP

	// get the runtime config
	runtimeConfig := runP.BaseConfig()

	// create the monitor plugin
	monP, err := pluginProvider.CreateMonitor(r.Monitor.Name, r.Name, runtimeConfig.Namespace, r.Runtime.Name)
	if err != nil {
		return nil, err
	}

	// configure the monitor plugin
	monP.Configure(r.Monitor.Config, sm.logger.ResetNamed("monitor-plugin"), sm.storage.CreatePluginStateStore(r, "monitor"))
	sm.monitorPlugin = monP

	// create the strategy plugin
	stratP, err := pluginProvider.CreateStrategy(r.Strategy.Name, monP)
	if err != nil {
		return nil, err
	}

	// configure the strategy plugin
	stratP.Configure(r.Strategy.Config, sm.logger.ResetNamed("strategy-plugin"), sm.storage.CreatePluginStateStore(r, "strategy"))
	sm.strategyPlugin = stratP

	// configure the webhooks
	for _, w := range r.Webhooks {
		wp, err := pluginProvider.CreateWebhook(w.Name)
		if err != nil {
			return nil, err
		}

		err = wp.Configure(w.Config, sm.logger.ResetNamed("webhooks-plugin"), sm.storage.CreatePluginStateStore(r, "webhooks"))
		if err != nil {
			return nil, err
		}

		sm.webhookPlugins = append(sm.webhookPlugins, wp)
	}

	// configure the post deployment tests
	if r.PostDeploymentTest != nil {
		testP, err := pluginProvider.CreatePostDeploymentTest(r.PostDeploymentTest.Name, releaserConfig.ConsulService, releaserConfig.Namespace, r.Runtime.Name, monP)
		if err != nil {
			return nil, err
		}

		err = testP.Configure(r.PostDeploymentTest.Config, sm.logger.ResetNamed("post-deployment-tests-plugin"), sm.storage.CreatePluginStateStore(r, "post-deployment-tests"))
		if err != nil {
			return nil, err
		}

		sm.testPlugin = testP
	}

	sm.logger.Debug("Current release state", "state", r.CurrentState())

	initialState := interfaces.StateStart
	if r.CurrentState() != "" {
		initialState = r.CurrentState()
	}

	f := fsm.NewFSM(
		initialState,
		fsm.Events{
			{Name: interfaces.EventConfigure, Src: []string{interfaces.StateStart, interfaces.StateIdle, interfaces.StateFail}, Dst: interfaces.StateConfigure},
			{Name: interfaces.EventConfigured, Src: []string{interfaces.StateConfigure}, Dst: interfaces.StateIdle},
			{Name: interfaces.EventDeploy, Src: []string{interfaces.StateIdle, interfaces.StateFail}, Dst: interfaces.StateDeploy},
			{Name: interfaces.EventDeployed, Src: []string{interfaces.StateDeploy}, Dst: interfaces.StateMonitor},
			{Name: interfaces.EventHealthy, Src: []string{interfaces.StateMonitor}, Dst: interfaces.StateScale},
			{Name: interfaces.EventScaled, Src: []string{interfaces.StateScale}, Dst: interfaces.StateMonitor},
			{Name: interfaces.EventComplete, Src: []string{interfaces.StateMonitor}, Dst: interfaces.StatePromote},
			{Name: interfaces.EventPromoted, Src: []string{interfaces.StatePromote}, Dst: interfaces.StateIdle},
			{Name: interfaces.EventUnhealthy, Src: []string{interfaces.StateMonitor}, Dst: interfaces.StateRollback},
			{Name: interfaces.EventComplete, Src: []string{interfaces.StateDeploy}, Dst: interfaces.StateIdle},
			{Name: interfaces.EventComplete, Src: []string{interfaces.StateRollback}, Dst: interfaces.StateIdle},
			{Name: interfaces.EventComplete, Src: []string{interfaces.StateDestroy}, Dst: interfaces.StateIdle},
			{Name: interfaces.EventFail, Src: []string{
				interfaces.StateStart,
				interfaces.StateConfigure,
				interfaces.StateIdle,
				interfaces.StateDeploy,
				interfaces.StateMonitor,
				interfaces.StateScale,
				interfaces.StatePromote,
				interfaces.StateRollback,
				interfaces.StateDestroy,
			}, Dst: interfaces.StateFail},
			{Name: interfaces.EventDestroy, Src: []string{
				interfaces.StateFail,
				interfaces.StateIdle,
				interfaces.StateConfigure,
				interfaces.StateDeploy,
				interfaces.StateMonitor,
				interfaces.StateScale,
				interfaces.StatePromote,
				interfaces.StateRollback,
			}, Dst: interfaces.StateDestroy},
		},
		fsm.Callbacks{
			"before_event":                       sm.logEvent(),
			"enter_" + interfaces.StateConfigure: sm.doConfigure(), // do the necessary work to setup the release
			"enter_" + interfaces.StateDeploy:    sm.doDeploy(),    // new version of the application has been deployed
			"enter_" + interfaces.StateMonitor:   sm.doMonitor(),   // start monitoring changes in the applications health
			"enter_" + interfaces.StateScale:     sm.doScale(),     // scale the release
			"enter_" + interfaces.StatePromote:   sm.doPromote(),   // promote the release to primary
			"enter_" + interfaces.StateRollback:  sm.doRollback(),  // rollback the deployment
			"enter_" + interfaces.StateDestroy:   sm.doDestroy(),   // remove everything and revert to vanilla state
			"enter_state":                        sm.enterState(),
			"leave_state":                        sm.leaveState(),
		},
	)

	sm.FSM = f

	return sm, nil
}

// Resume the state machine
func (s *StateMachine) Resume() error {
	switch s.CurrentState() {
	case interfaces.StateMonitor:
		s.SetState(interfaces.StateDeploy)
		s.Event(interfaces.EventDeployed)
	}

	return nil
}

// Configure triggers the EventConfigure state
func (s *StateMachine) Configure() error {
	return s.Event(interfaces.EventConfigure)
}

// Deploy triggers the EventDeploy state
func (s *StateMachine) Deploy() error {
	return s.Event(interfaces.EventDeploy)
}

// Destroy triggers the event Destroy state
func (s *StateMachine) Destroy() error {
	return s.Event(interfaces.EventDestroy)
}

// CurrentState returns the current state of the machine
func (s *StateMachine) CurrentState() string {
	return s.FSM.Current()
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
		s.release.UpdateState(e.FSM.Current())

		err := s.storage.UpsertRelease(s.release)
		if err != nil {
			s.logger.Error("Unable to upsert release", "name", s.release.Name, "error", err)
		}
	}
}

func (s *StateMachine) leaveState() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Log state", "event", e.Event, "state", e.FSM.Current())

		// when we leave the state call the timing done function
		if s.metricsDone != nil {
			if e.Err != nil {
				s.metricsDone(http.StatusInternalServerError)
				return
			}

			s.metricsDone(http.StatusOK)
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
			err := s.releaserPlugin.Setup(ctx, s.runtimePlugin.PrimarySubsetFilter(), s.runtimePlugin.CandidateSubsetFilter())
			if err != nil {
				s.logger.Error("Configure completed with error", "error", err)

				s.callWebhooks(s.webhookPlugins, "Configure release failed", interfaces.StateConfigure, interfaces.EventFail, 0, 100, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// updating consul configuration is an asynchronous process, it is possible
			// that a deployment can be removed before the data plane has updated its
			// configuration. this can cause issues where requests are sent to service instances
			// that do not exist.
			//
			// since we can not exactly determine when the state has converged in the data plane
			// wait an arbitrary period of time.
			time.Sleep(stepDelay)

			// if a deployment already exists copy this to the primary
			status, err := s.runtimePlugin.InitPrimary(ctx, s.release.Name)
			if err != nil {
				s.logger.Error("Configure completed with error", "status", status, "error", err)

				s.callWebhooks(s.webhookPlugins, "Configure release failed", interfaces.StateConfigure, interfaces.EventFail, 0, 100, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// if we created a new primary, wait till all instances are healthy then scale all traffic to it
			if status == interfaces.RuntimeDeploymentUpdate {
				err = s.releaserPlugin.WaitUntilServiceHealthy(ctx, s.runtimePlugin.PrimarySubsetFilter())
				if err != nil {
					s.logger.Error("New Primary deployment not healthy", "error", err)

					s.callWebhooks(s.webhookPlugins, "Configure release failed", interfaces.StateConfigure, interfaces.EventFail, 0, 100, err)
					e.FSM.Event(interfaces.EventFail)
					return
				}

				err = s.releaserPlugin.Scale(ctx, 0)
				if err != nil {
					s.logger.Error("Configure completed with error", "error", err)

					s.callWebhooks(s.webhookPlugins, "Configure release failed", interfaces.StateConfigure, interfaces.EventFail, 0, 100, err)
					e.FSM.Event(interfaces.EventFail)
					return
				}

				// we can't determine when the configuration is synced to the proxy, wait an arbitrary period of time
				time.Sleep(stepDelay * 4)

				// remove the candidate
				err = s.runtimePlugin.RemoveCandidate(ctx)
				if err != nil {
					s.logger.Error("Configure completed with error", "error", err)

					s.callWebhooks(s.webhookPlugins, "Configure release failed", interfaces.StateConfigure, interfaces.EventFail, 100, 0, err)
					e.FSM.Event(interfaces.EventFail)
					return
				}
			}

			s.logger.Debug("Configure completed successfully")

			s.callWebhooks(s.webhookPlugins, "Configure release succeeded", interfaces.StateConfigure, interfaces.EventConfigured, 100, 0, nil)
			e.FSM.Event(interfaces.EventConfigured)
		}()
	}
}

func (s *StateMachine) doDeploy() func(e *fsm.Event) {
	return func(e *fsm.Event) {
		s.logger.Debug("Deploy", "state", e.FSM.Current())
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

		go func() {
			// wait a few seconds as deploy is called before the new deployment is admitted to the server
			time.Sleep(stepDelay)

			// clean up resources if we finish before timeout
			defer cancel()

			// Create a primary if one does not exist
			status, err := s.runtimePlugin.InitPrimary(ctx, s.release.Name)
			if err != nil {
				s.logger.Error("Deploy completed with error", "error", err)

				s.callWebhooks(s.webhookPlugins, "New Deployment failed", interfaces.StateDeploy, interfaces.EventFail, 100, 0, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			err = s.releaserPlugin.WaitUntilServiceHealthy(ctx, s.runtimePlugin.PrimarySubsetFilter())
			if err != nil {
				s.logger.Error("Configure completed with error", "error", err)

				s.callWebhooks(s.webhookPlugins, "New Deployment failed", interfaces.StateDeploy, interfaces.EventFail, 100, 0, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// now the primary has been created send 100 of traffic there
			err = s.releaserPlugin.Scale(ctx, 0)
			// work has failed, raise the failed event
			if err != nil {
				s.logger.Error("Deploy completed with error", "error", err)

				s.callWebhooks(s.webhookPlugins, "New Deployment failed", interfaces.StateDeploy, interfaces.EventFail, 100, 0, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// if we created a primary this is the first deploy, no need to execute the strategy
			if status == interfaces.RuntimeDeploymentUpdate {
				s.logger.Debug("Deploy completed, created primary, waiting for next candidate deployment")

				// updating consul configuration is an asynchronous process, it is possible
				// that a deployment can be removed before the data plane has updated its
				// configuration. this can cause issues where requests are sent to service instances
				// that do not exist.
				//
				// since we can not exactly determine when the state has converged in the data plane
				// wait an arbitrary period of time.
				time.Sleep(stepDelay * 4)

				// remove the candidate and wait for the next deployment
				err = s.runtimePlugin.RemoveCandidate(ctx)
				if err != nil {
					s.logger.Error("Deploy completed with error", "error", err)

					s.callWebhooks(s.webhookPlugins, "New deployment failed", interfaces.StateDeploy, interfaces.EventFail, 100, 0, err)
					e.FSM.Event(interfaces.EventFail)
					return
				}

				s.callWebhooks(s.webhookPlugins, "New deployment succeeded", interfaces.StateDeploy, interfaces.EventComplete, 100, 0, nil)
				e.FSM.Event(interfaces.EventComplete)
				return
			}

			// new deployment run the strategy
			s.logger.Debug("Deploy completed, executing strategy")
			s.callWebhooks(s.webhookPlugins, "New deployment succeeded, executing strategy", interfaces.StateDeploy, interfaces.EventDeployed, 100, 0, nil)
			e.FSM.Event(interfaces.EventDeployed)
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

			// run the post deployment tests if we have any
			if s.testPlugin != nil {
				s.logger.Debug("Executing post deployment tests")
				err := s.testPlugin.Execute(ctx, s.runtimePlugin.BaseState().CandidateName)

				if err != nil {
					// post deployment tests have failed rollback
					s.logger.Error("Post deployment tests completed with error", "error", err)

					s.callWebhooks(
						s.webhookPlugins,
						"post deployment tests failed",
						interfaces.StateMonitor,
						interfaces.EventFail,
						s.strategyPlugin.GetPrimaryTraffic(),
						s.strategyPlugin.GetCandidateTraffic(),
						err,
					)

					e.FSM.Event(interfaces.EventUnhealthy)
					return
				}
			}

			result, traffic, err := s.strategyPlugin.Execute(ctx, s.runtimePlugin.BaseState().CandidateName)

			// strategy has failed with an error
			if err != nil {
				s.logger.Error("Monitor completed with error", "error", err)

				s.callWebhooks(
					s.webhookPlugins,
					"Monitoring deployment failed",
					interfaces.StateMonitor,
					interfaces.EventFail,
					s.strategyPlugin.GetPrimaryTraffic(),
					s.strategyPlugin.GetCandidateTraffic(),
					err,
				)

				e.FSM.Event(interfaces.EventFail)
			}

			// strategy returned a response
			switch result {
			// when the strategy reports a healthy deployment
			case interfaces.StrategyStatusSuccess:
				// send the traffic with the healthy event so that it can be used for scaling
				s.logger.Debug("Monitor checks completed, candidate healthy")

				e.FSM.Event(interfaces.EventHealthy, traffic)

			// the strategy has completed the roll out promote the deployment
			case interfaces.StrategyStatusComplete:
				s.logger.Debug("Monitor checks completed, strategy complete")

				e.FSM.Event(interfaces.EventComplete)

			// the strategy has reported that the deployment is unhealthy, rollback
			case interfaces.StrategyStatusFailed:
				s.logger.Debug("Monitor checks completed, candidate unhealthy")

				s.callWebhooks(
					s.webhookPlugins,
					"Monitor deployment failed",
					interfaces.StateMonitor,
					interfaces.EventUnhealthy,
					s.strategyPlugin.GetPrimaryTraffic(),
					s.strategyPlugin.GetCandidateTraffic(),
					err,
				)

				e.FSM.Event(interfaces.EventUnhealthy)
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

				e.FSM.Event(interfaces.EventFail)
				return
			}

			traffic := e.Args[0].(int)

			err := s.releaserPlugin.Scale(ctx, traffic)
			if err != nil {
				s.logger.Error("Scale completed with error", "error", err)

				s.callWebhooks(
					s.webhookPlugins,
					"Scaling deployment failed",
					interfaces.StateMonitor,
					interfaces.EventFail,
					s.strategyPlugin.GetPrimaryTraffic(),
					s.strategyPlugin.GetCandidateTraffic(),
					err,
				)

				e.FSM.Event(interfaces.EventFail)
				return
			}

			s.logger.Debug("Scale completed successfully")

			s.callWebhooks(
				s.webhookPlugins,
				"Scaling deployment succeeded",
				interfaces.StateMonitor,
				interfaces.EventScaled,
				s.strategyPlugin.GetPrimaryTraffic(),
				s.strategyPlugin.GetCandidateTraffic(),
				nil,
			)

			e.FSM.Event(interfaces.EventScaled)
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
				s.callWebhooks(s.webhookPlugins, "Promoting candidate failed", interfaces.StatePromote, interfaces.EventFail, 0, 100, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// updating consul configuration is an asynchronous process, it is possible
			// that a deployment can be removed before the data plane has updated its
			// configuration. this can cause issues where requests are sent to service instances
			// that do not exist.
			//
			// since we can not exactly determine when the state has converged in the data plane
			// wait an arbitrary period of time.
			time.Sleep(stepDelay)

			// promote the candidate to primary
			_, err = s.runtimePlugin.PromoteCandidate(ctx)
			if err != nil {
				s.callWebhooks(s.webhookPlugins, "Promoting candidate failed", interfaces.StatePromote, interfaces.EventFail, 0, 100, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// check consul to ensure that the promoted deployment is healthy
			err = s.releaserPlugin.WaitUntilServiceHealthy(ctx, s.runtimePlugin.PrimarySubsetFilter())
			if err != nil {
				s.logger.Error("Promote completed with error", "error", err)

				s.callWebhooks(s.webhookPlugins, "Promoting candidate failed", interfaces.StatePromote, interfaces.EventFail, 0, 100, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// scale all traffic to the primary
			err = s.releaserPlugin.Scale(ctx, 0)
			if err != nil {
				s.callWebhooks(s.webhookPlugins, "Promoting candidate failed", interfaces.StatePromote, interfaces.EventFail, 0, 100, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			time.Sleep(stepDelay * 4)

			// scale down the canary
			err = s.runtimePlugin.RemoveCandidate(ctx)
			if err != nil {
				s.callWebhooks(s.webhookPlugins, "Promoting candidate failed", interfaces.StatePromote, interfaces.EventFail, 100, 0, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			s.callWebhooks(s.webhookPlugins, "Promoting candidate to primary succeeded", interfaces.StatePromote, interfaces.EventPromoted, 100, 0, err)
			e.FSM.Event(interfaces.EventPromoted)
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
				e.FSM.Event(interfaces.EventFail)

				s.callWebhooks(
					s.webhookPlugins,
					"Rolling back deployment failed",
					interfaces.StateRollback,
					interfaces.EventFail,
					s.strategyPlugin.GetPrimaryTraffic(),
					s.strategyPlugin.GetCandidateTraffic(),
					err,
				)

				return
			}

			// updating consul configuration is an asynchronous process, it is possible
			// that a deployment can be removed before the data plane has updated its
			// configuration. this can cause issues where requests are sent to service instances
			// that do not exist.
			//
			// since we can not exactly determine when the state has converged in the data plane
			// wait an arbitrary period of time.
			time.Sleep(stepDelay * 4)

			// scale down the canary
			err = s.runtimePlugin.RemoveCandidate(ctx)
			if err != nil {
				s.callWebhooks(s.webhookPlugins, "Rolling back deployment failed", interfaces.StateRollback, interfaces.EventFail, 100, 0, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			s.callWebhooks(s.webhookPlugins, "Deployment rolled back", interfaces.StateRollback, interfaces.EventComplete, 100, 0, err)
			e.FSM.Event(interfaces.EventComplete)
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
				s.callWebhooks(s.webhookPlugins, "Remove release failed", interfaces.StateDestroy, interfaces.EventFail, 100, 0, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// ensure that the original version is healthy in consul before progressing
			err = s.releaserPlugin.WaitUntilServiceHealthy(ctx, s.runtimePlugin.CandidateSubsetFilter())
			if err != nil {
				s.logger.Error("Configure completed with error", "error", err)

				s.callWebhooks(s.webhookPlugins, "Remove release failed", interfaces.StateDestroy, interfaces.EventFail, 100, 0, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// scale all traffic to the candidate
			err = s.releaserPlugin.Scale(ctx, 100)
			if err != nil {
				s.callWebhooks(s.webhookPlugins, "Remove release failed", interfaces.StateDestroy, interfaces.EventFail, 100, 0, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// updating consul configuration is an asynchronous process, it is possible
			// that a deployment can be removed before the data plane has updated its
			// configuration. this can cause issues where requests are sent to service instances
			// that do not exist.
			//
			// since we can not exactly determine when the state has converged in the data plane
			// wait an arbitrary period of time.
			time.Sleep(stepDelay * 4)

			// destroy the primary
			err = s.runtimePlugin.RemovePrimary(ctx)
			if err != nil {
				s.callWebhooks(s.webhookPlugins, "Remove release failed", interfaces.StateDestroy, interfaces.EventFail, 0, 100, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			// remove the consul config
			err = s.releaserPlugin.Destroy(ctx)
			if err != nil {
				s.callWebhooks(s.webhookPlugins, "Remove release failed", interfaces.StateDestroy, interfaces.EventFail, 0, 100, err)
				e.FSM.Event(interfaces.EventFail)
				return
			}

			s.callWebhooks(s.webhookPlugins, "Remove release succeeded", interfaces.StateDestroy, interfaces.EventComplete, 0, 100, err)
			e.FSM.Event(interfaces.EventComplete)
		}()
	}
}

// callWebhooks calls the defined webhooks, in the event of failure this function will log an error
// but does not interupt flow
func (s *StateMachine) callWebhooks(wh []interfaces.Webhook, title, state, result string, primaryTraffic, candidateTraffic int, err error) {
	for _, w := range wh {
		s.logger.Debug("Calling webhook", "title", title)

		errString := ""
		if err != nil {
			errString = err.Error()
		}

		message := interfaces.WebhookMessage{
			Title:            title,
			Name:             s.release.Name,
			Namespace:        s.release.Namespace,
			Outcome:          result,
			State:            state,
			PrimaryTraffic:   primaryTraffic,
			CandidateTraffic: candidateTraffic,
			Error:            errString,
		}

		err := w.Send(message)
		if err != nil {
			s.logger.Error("Unable to call webhook", "title", title, "error", err)
		}
	}
}
