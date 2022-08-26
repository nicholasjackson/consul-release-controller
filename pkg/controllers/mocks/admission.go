package mocks

import (
	"context"

	"github.com/nicholasjackson/consul-release-controller/pkg/controllers"
	"github.com/stretchr/testify/mock"
)

type Admission struct {
	mock.Mock
}

func (a *Admission) Check(
	ctx context.Context,
	name string,
	namespace string,
	labels map[string]string,
	version string,
	runtime string) (controllers.AdmissionResponse, error) {

	args := a.Called(ctx, name, namespace, labels, version, runtime)

	return args.Get(0).(controllers.AdmissionResponse), args.Error(1)
}
