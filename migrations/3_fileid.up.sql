ALTER TABLE public.file
    ADD COLUMN file_id uuid;

ALTER TABLE public.file
    ADD COLUMN p_file_id uuid;

update file set file_id=id,p_file_id=p_id;

ALTER TABLE public.file
    ALTER COLUMN file_id SET NOT NULL;