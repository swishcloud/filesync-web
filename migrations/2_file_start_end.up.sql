ALTER TABLE public.file
    ALTER COLUMN name TYPE character varying (100);

ALTER TABLE public.file
    ADD COLUMN start integer;

ALTER TABLE public.file
    ADD COLUMN "end" integer;

update file set start=0;

ALTER TABLE public.file
    ALTER COLUMN start SET NOT NULL;

ALTER TABLE public.server_file DROP COLUMN insert_time;

ALTER TABLE public.server_file
    ADD COLUMN insert_time timestamp without time zone;
    
update server_file set insert_time=now() at time zone 'utc';

ALTER TABLE public.server_file
    ALTER COLUMN insert_time SET NOT NULL;

-- PROCEDURE: public.delete_file_or_directory(uuid)

-- DROP PROCEDURE public.delete_file_or_directory(uuid);

CREATE OR REPLACE PROCEDURE public.delete_file_or_directory(
	in_id uuid)
LANGUAGE 'plpgsql'

AS $BODY$
DECLARE
max_r file.start%type:=-1;
BEGIN
LOCK TABLE file IN SHARE ROW EXCLUSIVE MODE;
select max(a."max") into max_r from (select max(start) from file union select max("end") from file) a;
WITH RECURSIVE CTE as(select * from file  where id=in_id
UNION ALL select f.* from file f inner join CTE on f.p_id=CTE.id )
update file set "end"=max_r+1 where id in(select id from CTE) and "end" is null;
END
$BODY$;
