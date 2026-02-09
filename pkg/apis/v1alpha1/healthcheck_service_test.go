package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheckService_GetHealthLiveness(t *testing.T) {
	t.Parallel()

	service := &HealthCheckService{}
	ret, err := service.GetHealthLiveness(t.Context())

	assert.NoError(t, err)
	assert.NotNil(t, ret)
	assert.Equal(t, "OK", ret.Status)
}

func TestHealthCheckService_GetHealthReadiness(t *testing.T) {
	t.Parallel()

	service := &HealthCheckService{}
	ret, err := service.GetHealthReadiness(t.Context())

	assert.NoError(t, err)
	assert.NotNil(t, ret)
	assert.Equal(t, "OK", ret.Status)
}
