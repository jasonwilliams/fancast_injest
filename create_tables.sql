CREATE TABLE podcasts (
	id	uuid not null,
	title	text,
	description	text,
	link	text,
	updated	text,
	updatedParsed  timestamp,
	published text,
	publishedParsed timestamp,
	author  jsonb,
	language	text,
	image jsonb,
	last_build_date	timestamp,
	copyright text,
	lastProcessed timestamp
);


CREATE TABLE podcast_episodes (
	id uuid not null,
	title text,
	description text,
	published text,
	publishedParsed text,
	image jsonb,
	author jsonb,
	enclosures jsonb,
	duration text,
	subtitle text,
	summary text
)