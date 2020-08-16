begin;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE public.partition
(
    id uuid NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    CONSTRAINT partition_pkey PRIMARY KEY (id)
);
CREATE TABLE public.commit
(
    id uuid NOT NULL,
    partition_id uuid NOT NULL,
    index bigint NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    CONSTRAINT commit_pkey PRIMARY KEY (id),
    CONSTRAINT "fk-commit-partition_id" FOREIGN KEY (partition_id)
        REFERENCES public.partition (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);
ALTER TABLE public.user
    ADD COLUMN partition_id uuid;
update public.user set partition_id='0bb4c750-deee-11ea-87d0-0242ac130003';
ALTER TABLE public.user
    ALTER COLUMN partition_id set not null;
insert into public.partition(id,insert_time)values('0bb4c750-deee-11ea-87d0-0242ac130003',cast(now() as timestamp without time zone));
ALTER TABLE public.file
    ADD COLUMN partition_id uuid;
update file set partition_id='0bb4c750-deee-11ea-87d0-0242ac130003';
ALTER TABLE public.file
    ALTER COLUMN partition_id set not null;
ALTER TABLE public.file
    ADD COLUMN start_commit_id uuid;
ALTER TABLE public.file
    ADD COLUMN end_commit_id uuid;
ALTER TABLE public.file
	ADD CONSTRAINT "fk-file-start_commit_id" FOREIGN KEY (start_commit_id)
        REFERENCES public.commit (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION;
ALTER TABLE public.file
	ADD CONSTRAINT "fk-file-end_commit_id" FOREIGN KEY (end_commit_id)
        REFERENCES public.commit (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION;

	
do $$ 
DECLARE
   curs RECORD;
   temp bigint:=-1;
   temp_start uuid;
   temp_end uuid;
begin 
   FOR curs IN  SELECT * FROM file LOOP
   raise notice 'start:% end:%', curs.start,curs."end";
   --loop through start
   select index into temp from commit where index=curs.start;
   IF temp is null then
    insert into commit(id,partition_id,index,insert_time)values( uuid_generate_v4(),'0bb4c750-deee-11ea-87d0-0242ac130003',curs.start,cast(now() as timestamp without time zone));
   else
    raise notice 'the commit with this start value already exists.';
   end if;
   select id into temp_start from commit where index=curs.start;
   if temp_start is null then
   RAISE EXCEPTION  'can not find commit with this start value.';
   end if;
   update file set start_commit_id=temp_start where id=curs.id;
   --loop through end
   if curs."end" is not null then
  	 	select index into temp from commit where index=curs."end";
   		IF temp is null then
    	insert into commit(id,partition_id,index,insert_time)values( uuid_generate_v4(),'0bb4c750-deee-11ea-87d0-0242ac130003',curs."end",cast(now() as timestamp without time zone));
		else
  		raise notice 'the commit with this end value  already exists.';
  		END IF;
		select id into temp_end from commit where index=curs."end";
  	    update file set end_commit_id=temp_end where id=curs.id;
   END IF;
   
   END LOOP ;
end $$;
ALTER TABLE public.file
    ALTER COLUMN start_commit_id set not null;
ALTER TABLE public.file
    DROP COLUMN start;
ALTER TABLE public.file
    DROP COLUMN "end";
end;