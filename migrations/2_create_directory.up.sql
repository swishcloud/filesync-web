BEGIN TRANSACTION;
-- Table: public.directory

-- DROP TABLE public.directory;

CREATE TABLE public.directory
(
    id uuid NOT NULL,
    name character varying(200) COLLATE pg_catalog."default" NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    update_time timestamp without time zone,
    delete_time timestamp without time zone,
    is_deleted boolean NOT NULL,
    p_id uuid,
    user_id uuid NOT NULL,
    CONSTRAINT directory_pkey PRIMARY KEY (id),
    CONSTRAINT "fk-p_id" FOREIGN KEY (p_id)
        REFERENCES public.directory (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT "fk-user_id" FOREIGN KEY (user_id)
        REFERENCES public.user (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

ALTER TABLE public."user"
    ALTER COLUMN name TYPE character varying (30) ;

INSERT INTO public."user"(
	id, name, insert_time, op_id)
	VALUES ('1bafce34-4f9c-11ea-b77f-2e728ce88125', 'temp user for migrating', LOCALTIMESTAMP, '3a3fba62-4f9c-11ea-b77f-2e728ce88125');

INSERT INTO public.directory(
	id, name, insert_time, update_time, is_deleted,p_id,user_id)
	VALUES ('6fb5dbaa-4f91-11ea-b77f-2e728ce88125', 'temp folder for migrating', LOCALTIMESTAMP,NULL, false, NULL,'1bafce34-4f9c-11ea-b77f-2e728ce88125');

ALTER TABLE public.file
    ADD COLUMN update_time timestamp without time zone;

ALTER TABLE public.file
    ADD COLUMN is_deleted boolean;

ALTER TABLE public.file
    ADD COLUMN delete_time timestamp without time zone;

ALTER TABLE public.file
    ADD COLUMN directory_id uuid;

UPDATE public.file set directory_id='6fb5dbaa-4f91-11ea-b77f-2e728ce88125';

ALTER TABLE public.file
    Alter COLUMN directory_id SET NOT NULL;
ALTER TABLE public.file
    ADD CONSTRAINT "file_fk-directory_id" FOREIGN KEY (directory_id)
    REFERENCES public.directory (id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION;

UPDATE public.file set is_deleted=false;

ALTER TABLE public.file
    Alter COLUMN is_deleted SET NOT NULL;

END TRANSACTION;