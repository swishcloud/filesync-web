--
-- TOC entry 202 (class 1259 OID 16442)
-- Name: file; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.file (
    id uuid NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    name character varying(50) NOT NULL,
    description character varying(100) NOT NULL,
    user_id uuid NOT NULL,
    file_info_id uuid,
    is_deleted boolean NOT NULL,
    is_hidden boolean NOT NULL,
    p_id uuid,
    type integer NOT NULL
);


--
-- TOC entry 201 (class 1259 OID 16427)
-- Name: file_block; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.file_block (
    id uuid NOT NULL,
    server_file_id uuid NOT NULL,
    p_id uuid,
    "end" bigint NOT NULL,
    start bigint NOT NULL,
    path uuid NOT NULL
);


--
-- TOC entry 199 (class 1259 OID 16402)
-- Name: file_info; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.file_info (
    id uuid NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    md5 character(36) NOT NULL,
    path text NOT NULL,
    user_id uuid NOT NULL,
    size bigint NOT NULL
);


--
-- TOC entry 203 (class 1259 OID 16482)
-- Name: log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.log (
    id uuid NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    file_id uuid NOT NULL,
    action integer NOT NULL,
    user_id uuid NOT NULL,
    p_id uuid,
    number bigint NOT NULL
);


--
-- TOC entry 198 (class 1259 OID 16397)
-- Name: server; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.server (
    id uuid NOT NULL,
    name character varying(20) NOT NULL,
    ip character varying(20) NOT NULL,
    port integer NOT NULL
);


--
-- TOC entry 200 (class 1259 OID 16412)
-- Name: server_file; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.server_file (
    id uuid NOT NULL,
    file_info_id uuid NOT NULL,
    insert_time time without time zone NOT NULL,
    uploaded_size bigint NOT NULL,
    is_completed boolean NOT NULL,
    server_id uuid NOT NULL
);


--
-- TOC entry 197 (class 1259 OID 16392)
-- Name: user; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public."user" (
    id uuid NOT NULL,
    name character varying(30) NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    op_id uuid NOT NULL
);


--
-- TOC entry 2780 (class 2606 OID 16431)
-- Name: file_block file_block_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_block
    ADD CONSTRAINT file_block_pkey PRIMARY KEY (id);


--
-- TOC entry 2774 (class 2606 OID 16409)
-- Name: file_info file_info-pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_info
    ADD CONSTRAINT "file_info-pkey" PRIMARY KEY (id);


--
-- TOC entry 2776 (class 2606 OID 16411)
-- Name: file_info file_info-uniquekey-md5; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_info
    ADD CONSTRAINT "file_info-uniquekey-md5" UNIQUE (md5);


--
-- TOC entry 2782 (class 2606 OID 16446)
-- Name: file file_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file
    ADD CONSTRAINT file_pkey PRIMARY KEY (id);


--
-- TOC entry 2784 (class 2606 OID 16486)
-- Name: log log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.log
    ADD CONSTRAINT log_pkey PRIMARY KEY (id);


--
-- TOC entry 2778 (class 2606 OID 16416)
-- Name: server_file server_file_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.server_file
    ADD CONSTRAINT server_file_pkey PRIMARY KEY (id);


--
-- TOC entry 2772 (class 2606 OID 16401)
-- Name: server server_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.server
    ADD CONSTRAINT server_pkey PRIMARY KEY (id);


--
-- TOC entry 2770 (class 2606 OID 16396)
-- Name: user user_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."user"
    ADD CONSTRAINT user_pkey PRIMARY KEY (id);


--
-- TOC entry 2787 (class 2606 OID 16432)
-- Name: file_block file_block_fk-p_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_block
    ADD CONSTRAINT "file_block_fk-p_id" FOREIGN KEY (p_id) REFERENCES public.file_block(id);


--
-- TOC entry 2788 (class 2606 OID 16437)
-- Name: file_block file_block_fk-server_file_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_block
    ADD CONSTRAINT "file_block_fk-server_file_id" FOREIGN KEY (server_file_id) REFERENCES public.server_file(id);


--
-- TOC entry 2789 (class 2606 OID 16447)
-- Name: file file_fk-file_info_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file
    ADD CONSTRAINT "file_fk-file_info_id" FOREIGN KEY (file_info_id) REFERENCES public.file_info(id);


--
-- TOC entry 2790 (class 2606 OID 16487)
-- Name: file file_fk-p_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file
    ADD CONSTRAINT "file_fk-p_id" FOREIGN KEY (p_id) REFERENCES public.file(id);


--
-- TOC entry 2791 (class 2606 OID 16492)
-- Name: log log_fk-file_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.log
    ADD CONSTRAINT "log_fk-file_id" FOREIGN KEY (file_id) REFERENCES public.file(id);


--
-- TOC entry 2793 (class 2606 OID 16502)
-- Name: log log_fk-p_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.log
    ADD CONSTRAINT "log_fk-p_id" FOREIGN KEY (p_id) REFERENCES public.file(id);


--
-- TOC entry 2792 (class 2606 OID 16497)
-- Name: log log_fk-user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.log
    ADD CONSTRAINT "log_fk-user_id" FOREIGN KEY (user_id) REFERENCES public."user"(id);


--
-- TOC entry 2785 (class 2606 OID 16417)
-- Name: server_file server_file_fk-file_info_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.server_file
    ADD CONSTRAINT "server_file_fk-file_info_id" FOREIGN KEY (file_info_id) REFERENCES public.file_info(id);


--
-- TOC entry 2786 (class 2606 OID 16422)
-- Name: server_file server_file_fk-server_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.server_file
    ADD CONSTRAINT "server_file_fk-server_id" FOREIGN KEY (server_id) REFERENCES public.server(id);