ALTER TABLE public."share"
    ADD COLUMN file_type integer;
UPDATE public."share" set file_type=-1;
ALTER TABLE public."share"
    ALTER COLUMN file_type set not null;