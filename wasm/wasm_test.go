package wasm

import (
	"context"
	_ "embed"
	"testing"
)

func TestRun(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			"success",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Run(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
