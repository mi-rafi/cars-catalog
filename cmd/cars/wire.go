//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/mi-raf/cars-catalog/internal/api"
	"github.com/mi-raf/cars-catalog/internal/database"
	"github.com/mi-raf/cars-catalog/internal/service"
	"github.com/mi-raf/cars-catalog/internal/swagger"
)

func initApp(ctx context.Context, cfg *config) (a *api.API, closer func(), err error) {
	wire.Build(
		initApiConfig,
		initPostgresConnection,
		initValidator,
		initHttpClientConfiguration,
		database.NewCarRepository,
		wire.Bind(new(database.CarRepository), new(*database.PgCarRepository)),
		swagger.NewAPIClient,
		service.NewCarService,
		api.New,
	)
	return nil, nil, nil
}
