-- SPDX-License-Identifier: Apache-2.0
-- Copyright 2024 Authors of SentryFlow
--
-- Schema for sentryflow-log Kong plugin

local typedefs = require "kong.db.schema.typedefs"

return {
  name = "sentryflow-log",
  fields = {
    { protocols = typedefs.protocols },
    { config = {
        type = "record",
        fields = {
          { http_endpoint = typedefs.url({
              required = true,
              description = "SentryFlow HTTP endpoint URL (e.g., http://sentryflow.sentryflow:8081/api/v1/events)" -- { sentryflow.sentryflow is default name of sentryflow deployment.namespace update if its different for the setup}
            })
          },
          { timeout = {
              type = "number",
              default = 10000,
              description = "Timeout in milliseconds for HTTP requests to SentryFlow"
            }
          },
          { keepalive = {
              type = "number",
              default = 60000,
              description = "Keepalive timeout in milliseconds"
            }
          },
          { queue = typedefs.queue },
        },
      },
    },
  },
}
