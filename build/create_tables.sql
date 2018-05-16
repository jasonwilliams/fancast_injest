CREATE TABLE podcasts (
    id  uuid PRIMARY KEY,
    title   text,
    title_tsv tsvector,
    description text,
    description_tsv tsvector,
    link    text,
    updated text,
    updated_parsed  timestamp,
    author  jsonb,
    language    text,
    image jsonb,
    itunes_ext jsonb,
    categories jsonb,
    feed_url text,
    copyright text,
    last_processed timestamp,
  active boolean
);


CREATE TABLE podcast_episodes (
    id uuid PRIMARY KEY,
    -- I need both ID and guid, ID will be the internal ID, guid is the field provided by the RSS Item
    -- When updating a podcast episode i will need to do a lookup on the GUID, i will only expect a single row so this should be unique
    guid text UNIQUE,
    title text,
    title_tsv tsvector,
    description text,
    description_tsv tsvector,
    published text,
    published_parsed timestamp,
    author jsonb,
    image jsonb,
    enclosures jsonb,
    digest text UNIQUE,
    itunes_ext jsonb,
    last_processed timestamp,
    parent uuid not null,
  active boolean
);

-- Indexes for podcast episodes
create index ON podcast_episodes (published_parsed);
create index ON podcast_episodes USING GIN (description_tsv);
create index ON podcast_episodes USING GIN (title_tsv);

create trigger tsvectorupdate_podcast_episodes_description before insert or update on podcast_episodes for each row execute procedure
tsvector_update_trigger(description_tsv, 'pg_catalog.english', description);

create trigger tsvectorupdate_podcast_episodes_title before insert or update on podcast_episodes for each row execute procedure
tsvector_update_trigger(title_tsv, 'pg_catalog.english', title);


-- Indexes for podcasts
create index ON podcasts (updated_parsed);
create index ON podcasts USING GIN (description_tsv);
create index ON podcasts USING GIN (title_tsv);
create trigger tsvectorupdate_podcasts_description before insert or update on podcasts for each row execute procedure
tsvector_update_trigger(description_tsv, 'pg_catalog.english', description);

create trigger tsvectorupdate_podcasts_title before insert or update on podcasts for each row execute procedure
tsvector_update_trigger(title_tsv, 'pg_catalog.english', title);
