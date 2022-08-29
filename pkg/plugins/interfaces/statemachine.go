package interfaces

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

type StateMachine interface {
	// Configure triggers the EventConfigure state
	Configure() error

	// Deploy triggers the EventDeploy state
	Deploy() error

	// Destroy triggers the event Destroy state
	Destroy() error

	// CurrentState returns the current state
	CurrentState() string

	// Resume the statemachine from the current state
	Resume() error
}
