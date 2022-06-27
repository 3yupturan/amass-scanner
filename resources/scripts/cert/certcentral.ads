-- Copyright 2022 Jeff Foley. All rights reserved.
-- Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

local json = require("json")

name = "CertCentral"
type = "cert"

function start()
    set_rate_limit(1)
end

function check()
    local c
    local cfg = datasrc_config()
    if cfg ~= nil then
        c = cfg.credentials
    end

    if (c ~= nil and c.username ~= nil and
        c.username ~= "" and c.key ~= nil and c.key ~= "") then
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

    if (c == nil or c.username == nil or
        c.username == "" or c.key == nil or c.key == "") then
        return
    end

    local body, err = json.encode({
        ['accountId']=c.username,
        ['domains']={domain},
    })
    if (err ~= nil and err ~= "") then
        return
    end

    local resp, err = request(ctx, {
        ['url']="https://daas.digicert.com/apicontroller/v1/scan/getSubdomains",
        ['method']="POST",
        ['data']=body,
        ['headers']={
            ['Content-Type']="application/json",
            ['X-DC-DEVKEY']=c.key,
        },
    })
    if (err ~= nil and err ~= "") then
        log(ctx, "vertical request to service failed: " .. err)
        return
    end

    local j = json.decode(resp)
    if (j == nil or j.error ~= nil) then
        return
    end

    for _, sub in pairs(j.data[1].subdomains) do
        new_name(ctx, sub)
    end
end
