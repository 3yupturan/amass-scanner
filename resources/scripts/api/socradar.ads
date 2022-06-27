-- Copyright 2022 Jeff Foley. All rights reserved.
-- Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

local json = require("json")

name = "SOCRadar"
type = "api"

function start()
    set_rate_limit(1)
end

function check()
    local c
    local cfg = datasrc_config()
    if cfg ~= nil then
        c = cfg.credentials
    end

    if (c ~= nil and c.key ~= nil and c.key ~= "") then
        return true
    end
    return false
end

function vertical(ctx, domain)
    local c
    local cfg = datasrc_config()
    if cfg ~= nil then
        c = cfg.credentials
    end

    if (c == nil or c.key == nil or c.key == "") then
        return
    end

    local resp, err = request(ctx, {['url']=build_url(domain, c.key)})
    if (err ~= nil and err ~= "") then
        log(ctx, "vertical request to service failed: " .. err)
        return
    end

    local j = json.decode(resp)
    if (j == nil or j.is_success ~= true) then
        return
    end

    for _, sub in pairs(j.data.subdomains) do
        new_name(ctx, sub)
    end
end

function build_url(domain, key)
    return "https://platform.socradar.com/api/threat/analysis?key=" .. key .. "&entity=" .. domain
end
