-- The versions a movie or an episode offers: every release the addons list
-- ("all"), or the best of each of 4K, 1080p and 720p ("qualities").
ALTER TABLE settings
    ADD COLUMN versions_offered text NOT NULL DEFAULT 'all';
