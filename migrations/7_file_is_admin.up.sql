ALTER TABLE public."user"
    ADD COLUMN is_admin boolean;
UPDATE public."user" set is_admin=false;
ALTER TABLE public."user"
    ALTER COLUMN is_admin set not null;
ALTER TABLE public.file DROP COLUMN p_id;