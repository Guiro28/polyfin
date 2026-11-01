-- How the videos of a server addon reach the apps: read by the app at
-- their source when it can (source), always through the server, which
-- fetches them from its own address (server), or through the server and
-- the proxy the settings name, as WARP, which the addon's own requests
-- take too (proxy). A user's own addons stay at their source.
ALTER TABLE addons
    ADD COLUMN route text NOT NULL DEFAULT 'source' CHECK (route IN ('source', 'server', 'proxy')),
    ADD CONSTRAINT addons_route_server CHECK (route = 'source' OR owner_id IS NULL);

-- The proxy the addons routed through one use: a socks5:// or http://
-- address, empty for none.
ALTER TABLE settings
    ADD COLUMN stream_proxy text NOT NULL DEFAULT '';
