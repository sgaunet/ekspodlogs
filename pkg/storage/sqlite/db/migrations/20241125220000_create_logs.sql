-- migrate:up

CREATE TABLE logs (
    id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
    profile character varying(50) NOT NULL,
    loggroup character varying(255) NOT NULL,
    event_time timestamp NOT NULL,
    namespace_name character varying(255) NOT NULL,
    pod_name character varying(255) NOT NULL,
    container_name character varying(255) NOT NULL,
    log TEXT NOT NULL
);

CREATE INDEX logs_event_time_idx ON logs (event_time);
CREATE INDEX logs_id_idx       ON logs (id) ;

-- migrate:down

DROP TABLE logs cascade;
