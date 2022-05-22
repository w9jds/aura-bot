CREATE TYPE ChannelType AS ENUM ('killboard', 'srp');

CREATE TABLE IF NOT EXISTS channels (
  channel_id TEXT NOT NULL,
  guild_id TEXT NOT NULL,
  type ChannelType NOT NULL,
  filter_id INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS watchers (
  channel_id TEXT NOT NULL,
  guild_id TEXT NOT NULL,
  filter_ids INTEGER[] NOT NULL,
  ignore_list INTEGER[],
  min_value NUMERIC,
  jump_distance INTEGER
);

CREATE TABLE IF NOT EXISTS names (
  id INTEGER NOT NULL,
  category TEXT NOT NULL,
  name TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_names ON names(id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels ON channels(channel_id, filter_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_watchers ON watchers(channel_id, filter_ids);