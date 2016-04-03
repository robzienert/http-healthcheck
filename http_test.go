package healthcheck

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalHealthStatusResponse(t *testing.T) {
	status := Status{
		Healthy: false,
		Statuses: ProviderStatuses{
			"foo": nil,
			"bar": errors.New("Not OK"),
		},
	}

	actual := MarshalHealthStatusResponse(status)
	expected := HealthStatusResponse{
		Status: map[string]string{
			"foo": "OK",
			"bar": "Not OK",
		},
	}

	assert.Equal(t, expected, actual)
}
