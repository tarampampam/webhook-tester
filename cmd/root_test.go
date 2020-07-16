package cmd

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions_Struct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		element         func() reflect.StructField
		wantCommand     string
		wantAlias       string
		wantDescription string
	}{
		{
			element: func() reflect.StructField {
				field, _ := reflect.TypeOf(Root{}).FieldByName("Version")
				return field
			},
			wantCommand:     "version",
			wantAlias:       "v",
			wantDescription: "Display application version",
		},
		{
			element: func() reflect.StructField {
				field, _ := reflect.TypeOf(Root{}).FieldByName("Serve")
				return field
			},
			wantCommand:     "serve",
			wantAlias:       "s",
			wantDescription: "Start application web server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.wantDescription, func(t *testing.T) {
			el := tt.element()
			if tt.wantCommand != "" {
				value, _ := el.Tag.Lookup("command")
				assert.Equal(t, tt.wantCommand, value)
			}

			if tt.wantAlias != "" {
				value, _ := el.Tag.Lookup("alias")
				assert.Equal(t, tt.wantAlias, value)
			}

			if tt.wantDescription != "" {
				value, _ := el.Tag.Lookup("description")
				assert.Equal(t, tt.wantDescription, value)
			}
		})
	}
}
