--migrations/000001_create_metric_table_up.sql
--Создание таблицы метрик

CREATE TABLE metric if not exists(
	id varchar NOT NULL,
	metric_type varchar NOT NULL,
	delta int NULL,
	value double precision NULL,
	CONSTRAINT newtable_pk PRIMARY KEY (id)
);