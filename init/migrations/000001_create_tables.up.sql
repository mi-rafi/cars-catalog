CREATE TABLE IF NOT EXISTS People (
    id_p bigserial PRIMARY KEY,
    name_p  varchar(20) NOT NULL CONSTRAINT non_empty_name CHECK(length(name_p)>0),
    surname_p varchar(60) NOT NULL CONSTRAINT non_empty_surname CHECK(length(surname_p)>0),
    patronymic_p varchar(40),
    CONSTRAINT unique_nsp1 UNIQUE (name_p, surname_p, patronymic_p)
);

CREATE TABLE IF NOT EXISTS Car (
    reg_num varchar(12) PRIMARY KEY,
    mark varchar(40) NOT NULL CONSTRAINT non_empty_mark CHECK(length(mark)>0),
    model varchar(40) NOT NULL CONSTRAINT non_empty_model CHECK(length(model)>0), 
    year_c integer,
    id_p integer REFERENCES People(id_p)
);
