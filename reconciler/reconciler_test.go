package reconciler

import (
	"fmt"
	"testing"
	"time"
)

func TestResult_ShouldReturn(t *testing.T) {
	type fields struct {
		Action       action
		RequeueAfter time.Duration
		Error        error
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "Empty result returns false",
			fields: fields{},
			want:   false,
		},
		{
			name: "continueAction returns false",
			fields: fields{
				Action: ContinueAction,
				Error:  nil,
			},
			want: false,
		},
		{
			name: "returnAction returns true",
			fields: fields{
				Action: ReturnAction,
			},
			want: true,
		},
		{
			name: "Error returns true",
			fields: fields{
				Error: fmt.Errorf("error"),
			},
			want: true,
		},
		{
			name: "returnAndRequeueAction true returns true",
			fields: fields{
				Action: ReturnAndRequeueAction,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Result{
				Action:       tt.fields.Action,
				RequeueAfter: tt.fields.RequeueAfter,
				Error:        tt.fields.Error,
			}
			if got := result.ShouldReturn(); got != tt.want {
				t.Errorf("Result.ShouldReturn() = %v, want %v", got, tt.want)
			}
		})
	}
}
