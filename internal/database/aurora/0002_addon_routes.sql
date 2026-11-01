-- How the videos of a server addon reach the apps: read by the app at
-- their source when it can (source), always through the server, which
-- fetches them from its own address (server), or through the server and
-- the proxy the settings name, as WARP, which the addon's own requests
-- take too (proxy). A user's own addons stay at their source.
ALTER TABLE addons
    ADD COLUMN IF NOT EXISTS route text NOT NULL DEFAULT 'source';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'addons_route_server') THEN
        ALTER TABLE addons
            ADD CONSTRAINT addons_route_server
            CHECK (route IN ('source', 'server', 'proxy') AND (route = 'source' OR owner_id IS NULL));
    END IF;
END $$;

-- The proxy the addons routed through one use: a socks5:// or http://
-- address, empty for none.
ALTER TABLE settings
    ADD COLUMN IF NOT EXISTS stream_proxy text NOT NULL DEFAULT '';
