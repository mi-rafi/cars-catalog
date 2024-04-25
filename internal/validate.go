package internal

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

const (
	fistCarYear = 1885
)

func LessThanCurrYearValidator(fl validator.FieldLevel) bool {

	if !fl.Field().CanInt() {
		return false
	}

	if v := fl.Field().Int(); v != 0 && (v < fistCarYear || v > int64(time.Now().Year())) {
		log.Debug().Int64("year", v).Int64("now", int64(time.Now().Year())).Msg("invalid year")
		return false
	}
	return true
}
