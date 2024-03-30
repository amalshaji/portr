--
-- PostgreSQL database dump
--

-- Dumped from database version 16.2 (Debian 16.2-1.pgdg120+2)
-- Dumped by pg_dump version 16.2 (Debian 16.2-1.pgdg120+2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: team; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.team (id, created_at, updated_at, name, slug) FROM stdin;
1       2024-03-30 14:13:32.763932+00   2024-03-30 14:13:32.763961+00   portr   portr
\.


--
-- Data for Name: user; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public."user" (id, created_at, updated_at, email, first_name, last_name, is_superuser) FROM stdin;
1       2024-03-30 14:13:28.931274+00   2024-03-30 14:13:28.931326+00   amalshajid@gmail.com    \N      \N      t
\.


--
-- Data for Name: team_users; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.team_users (id, created_at, updated_at, secret_key, role, team_id, user_id) FROM stdin;
1       2024-03-30 14:13:32.766736+00   2024-03-30 14:13:32.766754+00   portr_gpb2sW2uWmN3TASvzI6PzAmuqem0VrKwc9o7      admin   1       1
\.


--
-- Data for Name: connection; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.connection (created_at, updated_at, id, type, subdomain, port, status, started_at, closed_at, created_by_id, team_id) FROM stdin;
\.


--
-- Data for Name: githubuser; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.githubuser (id, github_id, github_access_token, github_avatar_url, user_id) FROM stdin;
1       18011385        ghu_uverybigrandomstringxB31LvIa        https://avatars.githubusercontent.com/u/18011385?v=4    1
\.


--
-- Data for Name: instancesettings; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.instancesettings (id, created_at, updated_at, smtp_enabled, smtp_host, smtp_port, smtp_username, smtp_password, from_address, add_user_email_subject, add_user_email_body, updated_by_id) FROM stdin;
1       2024-03-30 14:12:43.843191+00   2024-03-30 14:12:43.843204+00   f       \N      \N      \N      \N      \N      You've been added to team {{teamName}} on Portr!        Hello {{email}}\n\nYou've been added to team "{{teamName}}" on Portr.\n\nGet started by signing in with your github account at {{dashboardUrl}}   \N
\.


--
-- Data for Name: session; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.session (id, created_at, updated_at, token, expires_at, user_id) FROM stdin;
1       2024-03-30 14:13:28.946702+00   2024-03-30 14:13:28.946732+00   XV6ZpcwMnvxuYEkdOTS0mbJIYDyy6Bx6        2024-04-06 14:13:28.945745+00   1
\.


--
-- Name: aerich_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.aerich_id_seq', 1, true);


--
-- Name: githubuser_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.githubuser_id_seq', 1, true);


--
-- Name: instancesettings_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.instancesettings_id_seq', 1, true);


--
-- Name: session_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.session_id_seq', 1, true);


--
-- Name: team_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.team_id_seq', 1, true);


--
-- Name: team_users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.team_users_id_seq', 1, true);


--
-- Name: user_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.user_id_seq', 1, true);


--
-- PostgreSQL database dump complete
--