CREATE TYPE ChannelType AS ENUM ('KillBoard', 'SRP');

CREATE TABLE IF NOT EXISTS channels (
  channel_id TEXT NOT NULL,
  guild_id TEXT NOT NULL,
  type ChannelType NOT NULL,
  group_id INTEGER
);

CREATE TABLE IF NOT EXISTS names (
  id INTEGER NOT NULL,
  category TEXT NOT NULL,
  name TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_names ON names(id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels ON channels(channel_id, group_id);