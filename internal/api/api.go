package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	_ "github.com/mi-raf/cars-catalog/docs"
	"github.com/mi-raf/cars-catalog/internal"
	mod "github.com/mi-raf/cars-catalog/internal/models"
	"github.com/mi-raf/cars-catalog/internal/service"
	"github.com/rs/zerolog/log"
	echoSwagger "github.com/swaggo/echo-swagger"
)

const (
	MAX_LIMIT    = 100
	MIN_LIMIT    = 5
	MIN_CAR_YEAR = 1885
)

type (
	Config struct {
		Addr string
	}

	API struct {
		e    *echo.Echo
		s    *service.CarServise
		addr string
	}

	Context struct {
		echo.Context
		Ctx context.Context
	}
)

func New(ctx context.Context, cfg *Config, s *service.CarServise) (*API, error) {
	e := echo.New()
	a := &API{
		s:    s,
		e:    e,
		addr: cfg.Addr,
	}

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &Context{
				Context: c,
				Ctx:     ctx,
			}
			return next(cc)
		}
	})

	e.Use(logger())
	e.GET("/health", healthCheck)
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.GET("/car", a.getCarsWithFilter)
	e.DELETE("/car/:regnum", a.deleteCar)
	e.PATCH("/car", a.updateCar)
	e.POST("/car", a.addCar)

	return a, nil
}

func logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			start := time.Now()

			err := next(c)
			stop := time.Now()

			log.Debug().
				Str("remote", req.RemoteAddr).
				Str("user_agent", req.UserAgent()).
				Str("method", req.Method).
				Str("path", c.Path()).
				Int("status", res.Status).
				Dur("duration", stop.Sub(start)).
				Str("duration_human", stop.Sub(start).String()).
				Msgf("called url %s", req.URL)
			return err
		}
	}
}

// healthcheck to check is server alive
// @Summary Show the status of server.
// @Description get the status of server.
// @Accept */*
// @Produce json
// @Success 200
// @Router /health [get]
func healthCheck(e echo.Context) error {
	return e.JSON(http.StatusOK, struct {
		Message string
	}{Message: "OK"})
}

type (
	CarJSON struct {
		RegNum string      `json:"regNum" `
		Mark   string      `json:"mark"`
		Model  string      `json:"model"`
		Year   int32       `json:"year,omitempty"`
		Owner  *PeopleJSON `json:"owner"`
	}

	PeopleJSON struct {
		Name       string `json:"name"`
		Surname    string `json:"surname"`
		Patronymic string `json:"patronymic,omitempty"`
	}

	RegNumRequestJSON struct {
		RegNums []string `json:"regNums"`
	}
)

// @Summary Get cars with filter
// @Description method to get some cars from database with filter and pagination. If filter is empty this method return all cars.
// @Produce json
// @Success 200 {object} CarJSON
// @Param limit query int false "limit of responce size"
// @Param offset query int false "offset of responce for database"
// @Param year query int false "car's filter param year"
// @Param reg_num query string false "car's filter param registration number"
// @Param mark query string false "car's filter param mark"
// @Param model query string false "car's filter param model"
// @Param name query string false "car's filter param owner's name"
// @Param surname query string false "car's filter param owner's surname"
// @Param patronymic query string false "car's filter param owner's patronymic"
// @Failure      400  {string}  string    "error"
// @Failure      500  {string}  string    "error"
// @Router /car [get]
func (a *API) getCarsWithFilter(e echo.Context) error {
	cc, err := getParentContext(e)
	if err != nil {
		log.Error().Err(err).Msg("can't get parent context in get")
		return err
	}

	limit, err := safeAtoi(e.QueryParam("limit"), func(i int) bool { return i > 0 })
	if err != nil {
		log.Debug().Err(err).Msg("incorrect limit")
		return err
	}
	limit = min(limit, MAX_LIMIT)
	limit = max(limit, MIN_LIMIT)

	offset, err := safeAtoi(e.QueryParam("offset"), func(i int) bool { return i >= 0 })
	if err != nil {
		log.Debug().Err(err).Msg("incorrect offset")
		return err
	}

	year, err := safeAtoi(e.QueryParam("year"), func(i int) bool { return i >= MIN_CAR_YEAR && i <= time.Now().Year() })
	if err != nil {
		log.Debug().Err(err).Msg("incorrect year")
		return err
	}

	filter := mod.CarFilter{
		RegNum:     e.QueryParam("reg_num"),
		Mark:       e.QueryParam("mark"),
		Model:      e.QueryParam("model"),
		Year:       int32(year),
		Name:       e.QueryParam("name"),
		Surname:    e.QueryParam("surname"),
		Patronymic: e.QueryParam("patronymic"),
	}

	cars, err := a.s.GetAll(cc.Ctx, filter, offset, limit)
	if err != nil {
		log.Error().Err(err).Msg("can't find cars")
		return echo.ErrInternalServerError
	}
	log.Debug().Interface("filter", filter).Msg("get cars with filter")
	carsJ := make([]CarJSON, 0, len(cars))
	for _, c := range cars {
		carsJ = append(carsJ, mapCarToJSON(&c))
	}
	return e.JSON(http.StatusOK, carsJ)

}

func safeAtoi(data string, validator func(int) bool) (int, error) {
	var res int
	if len(data) < 1 {
		return 0, nil
	}
	res, err := strconv.Atoi(data)
	if err != nil {
		log.Debug().Err(err).Str("data", data).Msg("can not parse int")
		return 0, echo.NewHTTPError(http.StatusBadRequest, "can not parse value: " + data)
	}
	if validator(res) {
		return res, nil
	}
	return 0, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("value %s is invalid", data))

}

// @Summary Add new cars.
// @Accept json
// @Produce json
// @Success 201
// @Param body body RegNumRequestJSON true "new car's registraton number"
// @Failure      400  {string}  string    "error"
// @Failure      500  {string}  string    "error"
// @Router /car [post]
func (a *API) addCar(e echo.Context) error {
	cc, err := getParentContext(e)
	if err != nil {
		log.Error().Err(err).Msg("can't get parent context in add")
		return err
	}

	regsJ := &RegNumRequestJSON{}
	err = e.Bind(regsJ)
	if err != nil {
		log.Debug().Err(err).Msg("can not unmarshall data")
		return echo.NewHTTPError(http.StatusBadRequest, "Incorrect data")
	}

	err = a.s.AddAll(cc.Ctx, regsJ.RegNums)
	if _, ok := err.(validator.ValidationErrors); ok {
		log.Debug().Err(err).Msg("Invalid data from client")
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid car data")
	}
	if _, ok := err.(*validator.InvalidValidationError); ok {
		log.Debug().Err(err).Msg("Invalid data from client, validation failure")
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid car data")
	}
	if e, ok := err.(internal.ClientError); ok {
		log.Debug().Err(err).Msg("Invalid response from client")
		return echo.NewHTTPError(e.Code, e.Msg)
	}
	if err != nil {
		log.Error().Err(err).Msg("can not add data")
		return echo.ErrInternalServerError
	}
	log.Debug().Interface("cars", regsJ).Msg("cars add to database")
	return e.NoContent(http.StatusCreated)

}

// @Summary Update new cars.
// @Accept json
// @Produce json
// @Success 200
// @Param request body CarJSON true "new car's version "
// @Failure      400  {string}  string    "error"
// @Failure      500  {string}  string    "error"
// @Router /car [patch]
func (a *API) updateCar(e echo.Context) error {
	cc, err := getParentContext(e)
	if err != nil {
		log.Error().Err(err).Msg("can't get parent context in update")
		return err
	}
	carJ := &CarJSON{Owner: &PeopleJSON{}}
	err = e.Bind(carJ)
	if (len(carJ.Owner.Name) < 1 && len(carJ.Owner.Surname) > 1) || (len(carJ.Owner.Name) > 1 && len(carJ.Owner.Surname) < 1) {
		log.Error().Msg("incoerrct name and surname")
		return echo.NewHTTPError(http.StatusBadRequest, "incoerrct name or surname")
	}
	if err != nil {
		log.Debug().Err(err).Msg("can not unmarshall data")
		return echo.NewHTTPError(http.StatusBadRequest, "can not unmarshall data")
	}

	car := mapJSONToCar(carJ)
	err = a.s.Update(cc.Ctx, &car)
	if err != nil {
		log.Error().Err(err).Msg("can not update data")
		return echo.ErrInternalServerError
	}
	log.Debug().Interface("car", car).Msg("update car")
	return e.NoContent(http.StatusOK)
}

// @Summary Delete car by registration namber.
// @Accept */*
// @Produce json
// @Success 200
// @Param reg_num path string true "car's param registration number"
// @Failure      400  {string}  string    "error"
// @Failure      500  {string}  string    "error"
// @Router /car [delete]
func (a *API) deleteCar(e echo.Context) error {
	cc, err := getParentContext(e)
	if err != nil {
		log.Error().Err(err).Msg("can't get parent context in delete")
		return err
	}

	regNum := e.Param("regnum")
	if len(regNum) < 1 {
		log.Error().Err(err).Msg("reg num is nil")
		return echo.NewHTTPError(http.StatusBadRequest, "incorrect registration number")
	}
	err = a.s.Delete(cc.Ctx, regNum)
	if err != nil {
		log.Error().Err(err).Str("mine regnum car", regNum).Msg("can't delete animal")
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	log.Debug().Str("reg num", regNum).Msg("delete car")
	return e.NoContent(http.StatusOK)
}

func getParentContext(e echo.Context) (*Context, error) {
	cc, ok := e.(*Context)
	if !ok {
		log.Error().Interface("type", reflect.TypeOf(e)).Msg("can't cast to custom context")
		return nil, echo.ErrInternalServerError
	}
	return cc, nil
}

func (a *API) Start() error {
	log.Debug().Msgf("listening on %v", a.addr)
	err := a.e.Start(a.addr)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *API) Close() error {
	log.Debug().Msg("Close API")
	return a.e.Close()
}

func mapJSONToCar(cJson *CarJSON) mod.CarDTO {
	log.Debug().Interface("car", cJson).Msg("map JSON to car")
	owner := mod.PeopleDTO{
		Name:       cJson.Owner.Name,
		Surname:    cJson.Owner.Surname,
		Patronymic: cJson.Owner.Patronymic,
	}
	car := mod.CarDTO{
		RegNum: cJson.RegNum,
		Mark:   cJson.Mark,
		Model:  cJson.Model,
		Year:   cJson.Year,
		Owner:  &owner,
	}
	return car
}

func mapCarToJSON(car *mod.CarDTO) CarJSON {
	log.Debug().Interface("car", car).Msg("map car to JSON")
	owner := PeopleJSON{
		Name:       car.Owner.Name,
		Surname:    car.Owner.Surname,
		Patronymic: car.Owner.Patronymic,
	}
	carJ := CarJSON{
		RegNum: car.RegNum,
		Mark:   car.Mark,
		Model:  car.Model,
		Year:   car.Year,
		Owner:  &owner,
	}
	return carJ
}
