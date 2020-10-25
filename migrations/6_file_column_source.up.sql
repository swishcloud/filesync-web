ALTER TABLE public.file
    ADD COLUMN source uuid;
ALTER TABLE public.file
    ADD CONSTRAINT "fk-source" FOREIGN KEY (source)
    REFERENCES public.file (id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION;