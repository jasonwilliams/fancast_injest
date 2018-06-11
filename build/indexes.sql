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
