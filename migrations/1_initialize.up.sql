-- Table: public."user"

-- DROP TABLE public."user";

CREATE TABLE public."user"
(
    id uuid NOT NULL,
    name character varying(20) COLLATE pg_catalog."default" NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    op_id uuid NOT NULL,
    CONSTRAINT user_pkey PRIMARY KEY (id)
)

-- Table: public.server

-- DROP TABLE public.server;

CREATE TABLE public.server
(
    id uuid NOT NULL,
    name character varying(20) COLLATE pg_catalog."default" NOT NULL,
    ip character varying(20) COLLATE pg_catalog."default" NOT NULL,
    port integer NOT NULL,
    CONSTRAINT server_pkey PRIMARY KEY (id)
)

-- Table: public.file_info

-- DROP TABLE public.file_info;

CREATE TABLE public.file_info
(
    id uuid NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    md5 character(36) COLLATE pg_catalog."default" NOT NULL,
    path text COLLATE pg_catalog."default" NOT NULL,
    user_id uuid NOT NULL,
    size bigint NOT NULL,
    CONSTRAINT "file_info-pkey" PRIMARY KEY (id),
    CONSTRAINT "file_info-uniquekey-md5" UNIQUE (md5)

)

-- Table: public.server_file

-- DROP TABLE public.server_file;

CREATE TABLE public.server_file
(
    id uuid NOT NULL,
    file_info_id uuid NOT NULL,
    insert_time time without time zone NOT NULL,
    uploaded_size bigint NOT NULL,
    is_completed boolean NOT NULL,
    server_id uuid NOT NULL,
    CONSTRAINT server_file_pkey PRIMARY KEY (id),
    CONSTRAINT "server_file_fk-file_info_id" FOREIGN KEY (file_info_id)
        REFERENCES public.file_info (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT "server_file_fk-server_id" FOREIGN KEY (server_id)
        REFERENCES public.server (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

-- Table: public.file_block

-- DROP TABLE public.file_block;

CREATE TABLE public.file_block
(
    id uuid NOT NULL,
    server_file_id uuid NOT NULL,
    p_id uuid,
    "end" bigint NOT NULL,
    start bigint NOT NULL,
    path uuid NOT NULL,
    CONSTRAINT file_block_pkey PRIMARY KEY (id),
    CONSTRAINT "file_block_fk-p_id" FOREIGN KEY (p_id)
        REFERENCES public.file_block (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT "file_block_fk-server_file_id" FOREIGN KEY (server_file_id)
        REFERENCES public.server_file (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

-- Table: public.file

-- DROP TABLE public.file;

CREATE TABLE public.file
(
    id uuid NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    name character varying(50) COLLATE pg_catalog."default" NOT NULL,
    description character varying(100) COLLATE pg_catalog."default" NOT NULL,
    user_id uuid NOT NULL,
    file_info_id uuid NOT NULL,
    CONSTRAINT file_pkey PRIMARY KEY (id),
    CONSTRAINT "file_fk-file_info_id" FOREIGN KEY (file_info_id)
        REFERENCES public.file_info (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)