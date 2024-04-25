package service

import (
	"context"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/mi-raf/cars-catalog/internal"
	"github.com/mi-raf/cars-catalog/internal/database"
	mod "github.com/mi-raf/cars-catalog/internal/models"
	"github.com/mi-raf/cars-catalog/internal/swagger"
	"github.com/rs/zerolog/log"
)

type (
	CarServise struct {
		r   database.CarRepository
		cli *swagger.APIClient
		v   *validator.Validate
	}
)

func NewCarService(r database.CarRepository, cli *swagger.APIClient, v *validator.Validate) *CarServise {
	log.Debug().Msg("create car service")
	return &CarServise{r: r, cli: cli, v: v}
}

func (c *CarServise) Delete(ctx context.Context, regNum string) error {
	log.Debug().Msg("delete car in service")
	return c.r.Delete(ctx, regNum)
}

func (c *CarServise) AddAll(ctx context.Context, regNums []string) error {
	var carArr []mod.CarDTO

	for _, r := range regNums {
		car, resp, err := c.cli.DefaultApi.InfoGet(ctx, r)
		log.Debug().Str("reg num", r).Msg("get information from client")

		if err != nil {
			log.Error().Err(err).Msg("can't get information from client")
			return mapClientError(r, resp, err)

		}
		addCar := mapCar(car)
		if err = c.v.Struct(addCar); err != nil {
			log.Error().Err(err).Msg("can't validate info from client")
			return err
		}
		carArr = append(carArr, addCar)
	}
	log.Debug().Interface("car array", carArr).Msg("validated cars from api")
	return c.r.Add(ctx, carArr)

}

func mapClientError(regNum string, resp *http.Response, err error) error {
	if _, ok := err.(swagger.GenericSwaggerError); ok {
		msg := "Internal error"
		switch resp.StatusCode {
		case 400:
			msg = "Incorect data for car: " + regNum
		case 404:
			msg = "Can not find car: " + regNum
		}
		return internal.ClientError{
			Code: resp.StatusCode,
			Err:  err,
			Msg:  msg,
		}
	}
	return err
}

func mapCar(c swagger.Car) mod.CarDTO {
	return mod.CarDTO{
		RegNum: c.RegNum,
		Mark:   c.Mark,
		Model:  c.Model,
		Year:   c.Year,
		Owner: &mod.PeopleDTO{
			Name:       c.Owner.Name,
			Surname:    c.Owner.Surname,
			Patronymic: c.Owner.Patronymic,
		},
	}
}

func (c *CarServise) Update(ctx context.Context, car *mod.CarDTO) error {
	err := c.v.Struct(car)
	if err != nil {
		log.Error().Err(err).Msg("can't validate for update")
		return err
	}
	log.Debug().Interface("car", car).Msg("update car")
	return c.r.Update(ctx, car)
}

func (c *CarServise) GetAll(ctx context.Context, filter mod.CarFilter, offset, limit int) ([]mod.CarDTO, error) {
	log.Debug().Interface("filter", filter).Msg("validated filter")
	return c.r.GetAll(ctx, filter, offset, limit)
}
