box.cfg{listen = "127.0.0.1:3301"}
box.schema.user.create('storage', {password='123', if_not_exists = true})
box.schema.user.grant('storage', 'super', nil, nil, {if_not_exists = true})
require('msgpack').cfg{encode_invalid_as_nil=true}