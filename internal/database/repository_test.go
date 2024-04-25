package database_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	//"github.com/go-delve/delve/pkg/dwarf/regnum"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mi-raf/cars-catalog/internal/database"

	mod "github.com/mi-raf/cars-catalog/internal/models"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type RepositoryTestSuite struct {
	suite.Suite
	r           database.CarRepository
	pgContainer *postgres.PostgresContainer
	ctx         context.Context
}

func (suite *RepositoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	var err error
	suite.pgContainer, err = postgres.RunContainer(suite.ctx,
		testcontainers.WithImage("postgres:15.3-alpine"),
		postgres.WithInitScripts(filepath.Join("..", "..", "testdata", "init-db.sql")),
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	suite.NoError(err)
	connStr, err := suite.pgContainer.ConnectionString(suite.ctx, "sslmode=disable")

	suite.NoError(err)
	p, err := pgxpool.New(suite.ctx, connStr)
	suite.NoError(err)
	suite.r, err = database.NewCarRepository(suite.ctx, p)
	suite.NoError(err)

	err = suite.pgContainer.CopyFileToContainer(suite.ctx, filepath.Join("..", "..", "testdata", "insert-cars.sql"), "/insert-cars.sql", int64(os.ModePerm.Perm()))
	suite.NoError(err)
	err = suite.pgContainer.CopyFileToContainer(suite.ctx, filepath.Join("..", "..", "testdata", "drop-cars.sql"), "/drop-cars.sql", int64(os.ModePerm.Perm()))
	suite.NoError(err)

}

func (suite *RepositoryTestSuite) SetupTest() {
	_, _, err := suite.pgContainer.Exec(suite.ctx, []string{"psql", "-U", "postgres", "-d", "test-db", "-f", "/insert-cars.sql"})
	suite.NoError(err)
}

func (suite *RepositoryTestSuite) TearDownTest() {
	_, _, err := suite.pgContainer.Exec(suite.ctx, []string{"psql", "-U", "postgres", "-d", "test-db", "-f", "/drop-cars.sql"})
	suite.NoError(err)
}

func (s *RepositoryTestSuite) TearDownSuite() {
	err := s.pgContainer.Terminate(s.ctx)
	s.NoError(err)
}

func (s *RepositoryTestSuite) TestGetAll() {
	filter := mod.CarFilter{}
	cars, err := s.r.GetAll(s.ctx, filter, 0, 10)
	s.NoError(err)
	s.NotNil(cars)
	s.Equal(6, len(cars))
}

func (s *RepositoryTestSuite) TestGetAllWithFilter() {
	filter := mod.CarFilter{
		RegNum: "aa000a00",
	}
	cars, err := s.r.GetAll(s.ctx, filter, 0, 10)
	s.NoError(err)
	s.NotNil(cars)
	s.Equal(1, len(cars))
}

func (s *RepositoryTestSuite) TestCreateCar() {
	//given
	expCar := &mod.CarDTO{
		RegNum: "cc332e10",
		Mark:   "BMW",
		Model:  "21trw",
		Year:   2020,
		Owner: &mod.PeopleDTO{
			Name:       "Fil",
			Surname:    "Foo",
			Patronymic: "Bar",
		}}

	expCarArr := []mod.CarDTO{*expCar}
	//when
	err := s.r.Add(s.ctx, expCarArr)
	//then
	s.NoError(err)

}

func (s *RepositoryTestSuite) TestCreateCarWihtoutPat() {
	//given
	expCar := &mod.CarDTO{
		RegNum: "cc336e10",
		Mark:   "BMW",
		Model:  "21trw",
		Year:   2020,
		Owner: &mod.PeopleDTO{
			Name:    "Fil",
			Surname: "Foo",
		}}

	expCarArr := []mod.CarDTO{*expCar}
	//when
	err := s.r.Add(s.ctx, expCarArr)
	//then
	s.NoError(err)

}

func (s *RepositoryTestSuite) TestCreateCarWihtoutYear() {
	//given
	expCar := &mod.CarDTO{
		RegNum: "cc336e10",
		Mark:   "BMW",
		Model:  "21trw",
		Owner: &mod.PeopleDTO{
			Name:    "Fil",
			Surname: "Foo",
		}}

	expCarArr := []mod.CarDTO{*expCar}
	//when
	err := s.r.Add(s.ctx, expCarArr)
	//then
	s.NoError(err)

}

func (s *RepositoryTestSuite) TestCreateCarWihtoutPat2() {
	//given
	expCar := &mod.CarDTO{
		RegNum: "qw234e123",
		Mark:   "BMW",
		Model:  "21trw",
		Year:   2020,
		Owner: &mod.PeopleDTO{
			Name:    "Fil",
			Surname: "Foo",
		}}

	expCarArr := []mod.CarDTO{*expCar}
	//when
	err := s.r.Add(s.ctx, expCarArr)
	//then
	s.NoError(err)
	c, err := s.r.GetAll(s.ctx, mod.CarFilter{RegNum: "qw234e123"}, 0, 10)
	s.NoError(err)
	s.Len(c, 1)

}

func (s *RepositoryTestSuite) TestCreateCarDuplicated() {
	//given
	expCar := &mod.CarDTO{
		RegNum: "rt123rt00",
		Mark:   "hot",
		Model:  "hot line",
		Year:   2020,
		Owner: &mod.PeopleDTO{
			Name:    "Ivan",
			Surname: "Scott",
		}}

	expCarArr := []mod.CarDTO{*expCar}
	//when
	err := s.r.Add(s.ctx, expCarArr)
	//then
	s.NoError(err)

}

func (s *RepositoryTestSuite) TestDeleteCar() {
	//given
	err := s.r.Delete(s.ctx, "rt123rt00")
	//then
	s.NoError(err)
	c, err := s.r.GetAll(s.ctx, mod.CarFilter{RegNum: "rt123rt00"}, 0, 10)
	s.NoError(err)
	s.Len(c, 0)
}

func (s *RepositoryTestSuite) TestUpdateCar() {
	owner := mod.PeopleDTO{
		Name:    "Ronald",
		Surname: "Wild",
	}
	carNew := mod.CarDTO{
		RegNum: "rt123rt00",
		Mark:   "lada",
		Model:  "s10",
		Owner:  &owner,
	}
	err := s.r.Update(s.ctx, &carNew)
	s.NoError(err)
}

func (s *RepositoryTestSuite) TestUpdateCarWithoutOwner() {
	owner := mod.PeopleDTO{}
	carNew := mod.CarDTO{
		RegNum: "rt123rt00",
		Mark:   "la",
		Model:  "s10",
		Owner:  &owner,
	}
	err := s.r.Update(s.ctx, &carNew)
	s.NoError(err)
}

func (s *RepositoryTestSuite) TestUpdateCarWithoutErrOwner() {
	owner := mod.PeopleDTO{
		Name: "Carl",
	}
	carNew := mod.CarDTO{
		RegNum: "rt123rt00",
		Mark:   "la",
		Model:  "s10",
		Owner:  &owner,
	}
	err := s.r.Update(s.ctx, &carNew)
	s.Error(err)
}

func TestCustomerRepoTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
