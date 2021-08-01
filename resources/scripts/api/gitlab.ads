-- Copyright 2017-2021 Jeff Foley. All rights reserved.
-- Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

name = "GitLab"
type = "api"

function start()
    setratelimit(1)
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

    local scopes = {"issues", "wiki_blobs", "blobs", "notes"}

    for _, s in pairs(scopes) do
        scrape(ctx, {
            url=apiurl(domain, s),
            headers={['PRIVATE-TOKEN']=c.key},
        })
    end
end

function apiurl(domain, scope)
    return "https://gitlab.com/api/v4/search?scope=" .. scope .. "&search=." .. domain
end
