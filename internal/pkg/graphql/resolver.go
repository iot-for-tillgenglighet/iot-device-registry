// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.
package graphql

import (
	"context"
)

type Resolver struct{}

func (r *deviceResolver) Name(ctx context.Context, obj *Device) (*string, error) {
	deviceName := "not implemented"
	return &deviceName, nil
}

func (r *entityResolver) FindDeviceByID(ctx context.Context, id string) (*Device, error) {
	return &Device{ID: id}, nil
}

func (r *queryResolver) Devices(ctx context.Context) ([]*Device, error) {
	return []*Device{}, nil
}

func (r *Resolver) Device() DeviceResolver { return &deviceResolver{r} }
func (r *Resolver) Entity() EntityResolver { return &entityResolver{r} }
func (r *Resolver) Query() QueryResolver   { return &queryResolver{r} }

type deviceResolver struct{ *Resolver }
type entityResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
