package workers

import "context"

// RunLoop is responsible for monitoring active deployments
type RunLoop struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// Run is a blocking function that monitors deployments
func (r *RunLoop) Run() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	r.ctx = ctx
	r.cancel = cancelFunc

	return nil
}

// Stop monitoring for deployments and exit the runloop
func (r *RunLoop) Stop() error {
	r.cancel()
	return nil
}

// IsRunning returns true when the run loop is running
func (r *RunLoop) IsRunning() bool {
	return r.ctx.Err() == nil
}
