package models

import (
	"github.com/looplab/fsm"
)

const (
	EventInactive    = "inactive"
	EventInitialize  = "initialize"
	EventInitialized = "initialized"
	EventDeploy      = "deploy"
	EventDeploying   = "deploying"
	EventDeployed    = "deployed"
	EventDestroy     = "destroy"
	EventDestroying  = "destroying"
	EventDestroyed   = "destroyed"
	EventMonitoring  = "monitoring"
	EventFail        = "fail"
	EventFailed      = "failed"
	EventCancel      = "cancel"
	EventCancelled   = "cancelled"
)

func newFSM(d *Deployment) *fsm.FSM {
	return fsm.NewFSM(
		EventInactive,
		fsm.Events{
			{Name: EventInitialize, Src: []string{EventInactive}, Dst: EventInitialized},
			{Name: EventDeploy, Src: []string{EventInitialized}, Dst: EventDeploying},
			{Name: EventDeployed, Src: []string{EventDeploying}, Dst: EventMonitoring},
			{Name: EventFail, Src: []string{EventDeploying, EventMonitoring, EventDestroying}, Dst: EventFailed},
			{Name: EventCancel, Src: []string{EventDeploying, EventMonitoring, EventDestroying}, Dst: EventCancelled},
		},
		fsm.Callbacks{
			"before_" + EventInitialize: d.initialize,
		},
	)
}
