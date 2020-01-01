-- Table: public.file

-- DROP TABLE public.file;

CREATE TABLE public.file
(
    id uuid NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    md5 character(32) COLLATE pg_catalog."default" NOT NULL,
    name text COLLATE pg_catalog."default" NOT NULL,
    user_id uuid NOT NULL,
    CONSTRAINT file_pkey PRIMARY KEY (id)
)
WITH (
    OIDS = FALSE
)
TABLESPACE pg_default;

ALTER TABLE public.file
    OWNER to filesync;