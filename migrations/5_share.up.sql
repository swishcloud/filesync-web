-- Table: public.share

-- DROP TABLE public.share;

CREATE TABLE public.share
(
    id uuid NOT NULL,
    token character varying(100) COLLATE pg_catalog."default" NOT NULL,
    path character varying COLLATE pg_catalog."default" NOT NULL,
    commit_id uuid NOT NULL,
    max_commit_id uuid NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    user_id uuid NOT NULL,
    partition_id uuid NOT NULL,
    CONSTRAINT share_pkey PRIMARY KEY (id),
    CONSTRAINT "fk-commit_id" FOREIGN KEY (commit_id)
        REFERENCES public.commit (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT "fk-max_commit_id" FOREIGN KEY (max_commit_id)
        REFERENCES public.commit (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT "fk-partition_id" FOREIGN KEY (partition_id)
        REFERENCES public.partition (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT "fk-user_id" FOREIGN KEY (user_id)
        REFERENCES public."user" (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);