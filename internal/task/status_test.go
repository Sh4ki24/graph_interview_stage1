package task

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apperrors "todoapp/internal/errors"
)

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		from, to Status
		wantErr  bool
	}{
		{StatusTodo, StatusProgress, false},
		{StatusTodo, StatusDone, true},   // can't skip progress
		{StatusTodo, StatusClose, false}, // close = soft delete, always allowed
		{StatusProgress, StatusDone, false},
		{StatusProgress, StatusTodo, true},
		{StatusDone, StatusClose, false},
		{StatusClose, StatusTodo, true}, // dead state
	}
	for _, tt := range tests {
		err := ValidateTransition(tt.from, tt.to)
		if tt.wantErr {
			assert.ErrorIs(t, err, apperrors.ErrInvalidTransition, "%s -> %s", tt.from, tt.to)
		} else {
			assert.NoError(t, err, "%s -> %s", tt.from, tt.to)
		}
	}
}

func TestSameStatusIsNoop(t *testing.T) {
	assert.NoError(t, ValidateTransition(StatusProgress, StatusProgress))
}

func TestValidStatus(t *testing.T) {
	assert.True(t, ValidStatus(StatusTodo))
	assert.True(t, ValidStatus(StatusProgress))
	assert.True(t, ValidStatus(StatusDone))
	assert.True(t, ValidStatus(StatusClose))
	assert.False(t, ValidStatus("archived")) // statuses must be consts; unknown rejected
}

func TestStatusIsActive(t *testing.T) {
	assert.True(t, StatusTodo.IsActive())
	assert.False(t, StatusClose.IsActive())
}
