INSERT INTO People(name_p, surname_p) VALUES('David', 'Scott');
INSERT INTO People(name_p, surname_p) VALUES('Ramazan', 'Tyrin');
INSERT INTO People(name_p, surname_p, patronymic_p) VALUES('Bob', 'Shakir', 'Valin');
INSERT INTO People(name_p, surname_p) VALUES('Ivan', 'Scott');

INSERT INTO Car(reg_num, mark, model, id_p) VALUES ('rt123rt00', 'hot', 'hot line', (SELECT id_p FROM People 
WHERE name_p = 'Ivan'));
INSERT INTO Car(reg_num, mark, model, year_c, id_p) VALUES ('aa000a00', 'hot', 'www', 1999, (SELECT id_p FROM People 
WHERE name_p = 'Ivan'));
INSERT INTO Car(reg_num, mark, model, year_c, id_p) VALUES ('bb123rt01', 'java', 'spring', 2018, (SELECT id_p FROM People 
WHERE name_p = 'David'));
INSERT INTO Car(reg_num, mark, model, year_c, id_p) VALUES ('ee523rt00', 'fer', '3c', 2002, (SELECT id_p FROM People 
WHERE name_p = 'Ramazan'));
INSERT INTO Car(reg_num, mark, model, year_c, id_p) VALUES ('rt666t00', 'winter', 'rainGosling', 2001, (SELECT id_p FROM People 
WHERE name_p = 'David'));
INSERT INTO Car(reg_num, mark, model, year_c, id_p) VALUES ('rt98457rtDS', 'cat', 'lion', 2010, (SELECT id_p FROM People 
WHERE name_p = 'Bob'));

