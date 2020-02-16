BEGIN TRANSACTION;
--table file
ALTER TABLE public.file
    ADD COLUMN is_hidden boolean;
update public.file set is_hidden=false;
ALTER TABLE public.file
    ALTER COLUMN is_hidden set not null;
--table directory
ALTER TABLE public.directory
    ADD COLUMN is_hidden boolean;
update public.directory set is_hidden=false;
ALTER TABLE public.directory
    ALTER COLUMN is_hidden set not null;
END TRANSACTION;