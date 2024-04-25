package database

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype/zeronull"
	"github.com/jackc/pgx/v5/pgxpool"
	mod "github.com/mi-raf/cars-catalog/internal/models"
)

const (
	delete              = "DELETE FROM Car WHERE reg_num = $1"
	searchRegNum        = "SELECT reg_num FROM Car WHERE reg_num = $1"
	selectOwnerID       = "SELECT id_p FROM People WHERE name_p = $1 AND surname_p = $2 AND CASE WHEN patronymic_p IS NULL THEN true ELSE patronymic_p = $3 END"
	insertOwner         = "INSERT INTO People (name_p, surname_p, patronymic_p) VALUES ($1, $2, $3) RETURNING id_p"
	insertCar           = "INSERT INTO Car (reg_num, mark, model, year_c, id_p ) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (reg_num) DO NOTHING"
	searchCarAllWithFil = `
	SELECT reg_num, mark, model, year_c, p.name_p AS name_p, p.surname_p AS surname_p, p.patronymic_p AS patronymic_p   
	FROM Car JOIN People AS p
	ON Car.id_p = p.id_p 
	   WHERE ($1::varchar IS NULL OR reg_num LIKE CONCAT('%%', $1::varchar, '%%')) AND
		($2::varchar IS NULL OR mark LIKE CONCAT('%%', $2::varchar, '%%'))  AND 
		($3::varchar IS NULL OR model LIKE CONCAT('%%', $3::varchar, '%%')) AND
		($4::integer IS NULL OR year_c = $4::integer) AND
		($5::varchar IS NULL OR p.name_p LIKE CONCAT('%%', $5::varchar, '%%')) AND
		($6::varchar IS NULL OR p.surname_p LIKE CONCAT('%%', $6::varchar, '%%')) AND
		($7::varchar IS NULL OR p.patronymic_p LIKE CONCAT('%%', $7::varchar, '%%'))
	ORDER BY reg_num
	LIMIT $8
	OFFSET $9`

	update = `UPDATE Car SET 
    			mark = COALESCE($2, mark),
    			model = COALESCE($3, model),
    			year_c = COALESCE($4, year_c),
    			id_p = COALESCE($5, id_p)
			WHERE reg_num = $1`
)

type (
	CarRepository interface {
		Delete(ctx context.Context, regNum string) error
		Add(ctx context.Context, cars []mod.CarDTO) error
		GetAll(ctx context.Context, filter mod.CarFilter, offset, limit int) ([]mod.CarDTO, error)
		Update(ctx context.Context, car *mod.CarDTO) error
	}

	PgCarRepository struct {
		pool *pgxpool.Pool
	}
)

func NewCarRepository(ctx context.Context, p *pgxpool.Pool) (*PgCarRepository, error) {

	return &PgCarRepository{pool: p}, nil
}

func (r *PgCarRepository) Close(ctx context.Context) error {
	r.pool.Close()
	return nil
}

func (r *PgCarRepository) Add(ctx context.Context, cars []mod.CarDTO) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("can't open transaction for add")
		return err
	}

	defer func() {
		err = tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) {
			log.Error().Err(err).Msg("Undefinded error in tx")
		}
	}()

	var owner *mod.PeopleDTO
	var ownerID int64

	for _, c := range cars {
		log.Debug().Interface("car", c).Msg("adding car")
		owner = c.Owner
		log.Debug().Interface("owner", owner).Msg("adding car owner")
		err = tx.QueryRow(ctx, selectOwnerID, owner.Name, owner.Surname, zeronull.Text(owner.Patronymic)).Scan(&ownerID)
		if err == pgx.ErrNoRows {
			log.Debug().Int64("owner id", ownerID).Msg("inserting new people")
			err = tx.QueryRow(ctx, insertOwner, owner.Name, owner.Surname, zeronull.Text(owner.Patronymic)).Scan(&ownerID)
		}
		if err != nil {
			log.Error().Err(err).Msg("error insert received")
			return err
		}

		_, err = tx.Exec(ctx, insertCar, c.RegNum, c.Mark, c.Model, zeronull.Int4(c.Year), ownerID)

		if err != nil {
			log.Error().Str("car's reg num", c.RegNum).Msg("can't insert car")
			return err
		}
		log.Debug().Str("car's reg num", c.RegNum).Msg("car insert to table")

	}
	return tx.Commit(ctx)
}

func (r *PgCarRepository) Delete(ctx context.Context, regNum string) error {
	if len(regNum) < 1 {
		log.Error().Msg("registration number is empty")
		return errors.New("regNum is empty")
	}
	log.Debug().Str("reg num", regNum).Msg("try delete")
	_, err := r.pool.Exec(ctx, delete, regNum)

	return err
}

func (r *PgCarRepository) GetAll(ctx context.Context, filter mod.CarFilter, offset, limit int) ([]mod.CarDTO, error) {

	rows, err := r.pool.Query(ctx, searchCarAllWithFil, zeronull.Text(filter.RegNum),
		zeronull.Text(filter.Mark), zeronull.Text(filter.Model), zeronull.Int4(filter.Year),
		zeronull.Text(filter.Name), zeronull.Text(filter.Surname), zeronull.Text(filter.Patronymic),
		zeronull.Int4(limit), zeronull.Int4(offset))
	if err == pgx.ErrNoRows {
		log.Debug().Msg("GetAll return 0 rows")
		return []mod.CarDTO{}, nil
	}
	if err != nil {
		log.Error().Err(err).Msg("can't return result getAll")
		return nil, err
	}

	cars := make([]mod.CarDTO, 0)

	for rows.Next() {
		c := mod.CarDTO{}
		owner := mod.PeopleDTO{}
		var yz zeronull.Int2
		var p zeronull.Text
		err = rows.Scan(&c.RegNum, &c.Mark, &c.Model, &yz, &owner.Name, &owner.Surname, &p)
		owner.Patronymic = string(p)
		c.Year = int32(yz)
		if err != nil {
			return nil, err
		}
		c.Owner = &owner
		cars = append(cars, c)
		log.Debug().Interface("car", c).Msg("get car with filter")
	}
	// check rows.Err() after the last rows.Next() :
	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("*** iteration error")
		return nil, err
	}

	return cars, nil
}

func (r *PgCarRepository) Update(ctx context.Context, car *mod.CarDTO) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("can't open transaction for update")
		return err
	}

	defer func() {
		err = tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) {
			log.Error().Err(err).Msg("Undefinded error in tx")
		}
	}()

	var ownerID int64
	if len(car.Owner.Name) > 1 && len(car.Owner.Surname) > 1 {
		owner := *car.Owner
		err = tx.QueryRow(ctx, selectOwnerID, owner.Name, owner.Surname, owner.Patronymic).Scan(&ownerID)
		if err == pgx.ErrNoRows {
			err = tx.QueryRow(ctx, insertOwner, owner.Name, owner.Surname, zeronull.Text(owner.Patronymic)).Scan(&ownerID)
			log.Debug().Interface("owner", owner).Msg("create new owner for update")
		}
	} else {
		if len(car.Owner.Name) > 1 || len(car.Owner.Surname) > 1 {
			log.Error().Msg("add new name and surname")
			return errors.New("add new name and surname")
		}
	}

	if err != nil {
		log.Error().Err(err).Msg("can't update.Check owner")
		return err
	}

	_, err = tx.Exec(ctx, update, car.RegNum, zeronull.Text(car.Mark), zeronull.Text(car.Model), zeronull.Int4(car.Year), zeronull.Int8(ownerID))
	if err != nil {
		log.Error().Err(err).Str("Reg num", car.RegNum).Msg("can't update car")
		return err
	}
	log.Debug().Str("reg num", car.RegNum).Msg("update car")
	return tx.Commit(ctx)

}
