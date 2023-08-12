package driver

import (
	"reflect"
	"sync"
	"testing"

	"github.com/smilelinkd/digitalbow-mapper/pkg/common"
	"gonum.org/v1/gonum/mat"
)

func TestDigitalbowClient_GetBowDataformat(t *testing.T) {
	type fields struct {
		Client       BowClient
		Status       common.DeviceStatus
		Movements    map[string]TrackData
		mu           sync.Mutex
		Transform_AU *mat.Dense
		Rotation_AU  *mat.Dense
	}
	type args struct {
		trackData  [4][4]float64
		matrixInit [4][4]float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []float64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &DigitalbowClient{
				Client:       tt.fields.Client,
				Status:       tt.fields.Status,
				Movements:    tt.fields.Movements,
				mu:           tt.fields.mu,
				Transform_AU: tt.fields.Transform_AU,
				Rotation_AU:  tt.fields.Rotation_AU,
			}
			if got := c.GetBowDataformat(tt.args.trackData, tt.args.matrixInit); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBowDataformat() = %v, want %v", got, tt.want)
			}
		})
	}
}
