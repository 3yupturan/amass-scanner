-- Copyright 2021 Jeff Foley. All rights reserved.
-- Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

name = "UKWebArchive"
type = "archive"

function start()
    setratelimit(3)
end

function vertical(ctx, domain)
    scrape(ctx, {['url']=buildurl(domain)})
end

function buildurl(domain)
    return "https://www.webarchive.org.uk/wayback/archive/*?matchType=domain&url=" .. domain
end
