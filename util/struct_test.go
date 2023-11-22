package util

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type TestStruct struct {
	A string `json:"a"`
	B string `json:"b"`
}

func TestStructToMap(t *testing.T) {
	type args struct {
		o any
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]any
		wantErr bool
	}{
		{
			name: "converts struct to map",
			args: args{
				o: &TestStruct{
					A: "1",
					B: "2",
				},
			},
			want:    map[string]any{"a": "1", "b": "2"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StructToMap(tt.args.o)
			if (err != nil) != tt.wantErr {
				t.Errorf("StructToMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); len(diff) > 0 {
				t.Errorf("TestMapToStruct() diff = %v", diff)
			}
		})
	}
}

func TestMapToStruct(t *testing.T) {
	type args struct {
		m map[string]any
		o any
	}
	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "converts map to struct",
			args: args{
				m: map[string]any{"a": "1", "b": "2"},
				o: &TestStruct{},
			},
			want: &TestStruct{
				A: "1",
				B: "2",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MapToStruct(tt.args.m, tt.args.o); (err != nil) != tt.wantErr {
				t.Errorf("MapToStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.args.o, tt.want); len(diff) > 0 {
				t.Errorf("TestMapToStruct() diff = %v", diff)
			}
		})
	}
}
