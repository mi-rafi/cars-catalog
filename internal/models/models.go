package internal

type (
	PeopleDTO struct {
		Id         int64
		Name       string `validate:"required"`
		Surname    string `validate:"required"`
		Patronymic string
	}

	CarDTO struct {
		RegNum string `validate:"required"`
		Mark   string `validate:"required"`
		Model  string `validate:"required"`
		Year   int32  `validate:"c-year"`
		Owner  *PeopleDTO
	}

	CarFilter struct {
		RegNum     string
		Mark       string
		Model      string
		Year       int32
		Name       string
		Surname    string
		Patronymic string
	}
)
