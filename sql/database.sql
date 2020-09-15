CREATE TYPE source_format AS ENUM ('OAI-PMH', 'Swedish', 'Oulu');
CREATE TYPE url_type AS ENUM ('normal', 'vapaakappale');

CREATE TABLE source (
       source_id        integer GENERATED ALWAYS AS IDENTITY UNIQUE,
       title            text NOT NULL,
       format           source_format,
       start_url        text NOT NULL,
       resume_url       text,
       priority         integer NOT NULL,
       email            text,
       description      text,
       source_type      url_type,
       url_pattern	text
);

CREATE TABLE urn2url (
       urn           text NOT NULL,
       url           text NOT NULL,
       source_id     INTEGER REFERENCES source(source_id),
       url_type      url_type,
       r_component   text
);

CREATE TABLE urnhistory (
       urn              text NOT NULL,
       r_component      text,
       url_old          text,
       url_new          text,
       url_type_old     url_type,
       url_type_new     url_type,
       harvest_time     timestamp with time zone,
       source_url       text NOT NULL
);
