package resource

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
)

func TestProperty_Reconcile(t *testing.T) {
	type args struct {
		target  map[string]any
		desired map[string]any
		logger  logr.Logger
	}
	tests := []struct {
		name       string
		p          Property
		args       args
		want       bool
		wantErr    bool
		wantTarget map[string]any
	}{
		{
			name: "",
			p:    "a.b.c",
			args: args{
				target:  map[string]any{"a": map[string]any{"b": map[string]any{"c": "value", "d": 1}}},
				desired: map[string]any{"a": map[string]any{"b": map[string]any{"c": "newValue"}}},
				logger:  logr.Discard(),
			},
			want:       true,
			wantErr:    false,
			wantTarget: map[string]any{"a": map[string]any{"b": map[string]any{"c": "newValue", "d": 1}}},
		},
		{
			name: "",
			p:    "a.b.c",
			args: args{
				target:  map[string]any{"a": map[string]any{}},
				desired: map[string]any{"a": map[string]any{"b": map[string]any{"c": "newValue"}}},
				logger:  logr.Discard(),
			},
			want:       true,
			wantErr:    false,
			wantTarget: map[string]any{"a": map[string]any{"b": map[string]any{"c": "newValue"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.p.Reconcile(tt.args.target, tt.args.desired, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("Property.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Property.Reconcile() = %v, want %v", got, tt.want)
			}
			if diff := cmp.Diff(tt.args.target, tt.wantTarget); len(diff) > 0 {
				t.Errorf("Property.Reconcile() diff  %v", diff)
			}
		})
	}
}

func Test_delta(t *testing.T) {
	g := gomega.NewWithT(t)
	g.Expect(delta(0, 0)).To(gomega.Equal(MissingInBoth))
	g.Expect(delta(0, 1)).To(gomega.Equal(MissingFromDesiredPresentInTarget))
	g.Expect(delta(1, 0)).To(gomega.Equal(PresentInDesiredMissingFromTarget))
	g.Expect(delta(1, 1)).To(gomega.Equal(PresentInBoth))
}
