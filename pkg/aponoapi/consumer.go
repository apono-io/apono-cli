package aponoapi

// ConsumedByAponoCli is the wire value this CLI sends in the `consumed_by`
// query param of the access-details endpoint. It claims the session as
// consumed by the CLI; if the BE echoes back a different value, another
// surface (Portal AD dialog, Slack, etc.) used the creds first and the user
// must reset before we can launch.
const ConsumedByAponoCli = "consumedByAponoCli"
