

function safeGet(value) {
    return (value === undefined || value === null) ? "" : value;
}

function getVar(r, name) {
    try {
        const v = r.variables[name];
        return (v === undefined || v === null) ? "" : v;
    } catch (err) {
        r.error(`getVar failed for ${name}: ${err}`);
        return "";
    }
}


async function rateLimitCheck(r) {

    r.log(`rateLimitCheck: entered`);

    const event = {
        source: safeGet(r.remoteAddress),
        destination: getVar(r, "server_addr"),
        method: r.method,
        path: r.uri,
    };

    try {
        const res = await ngx.fetch(
            "http://192.168.29.112:8082/allowed",
            {
                method: "POST",
                body: JSON.stringify(event),
                headers: {
                    "Content-Type": "application/json"
                },
                timeout: 20
            }
        );

        r.log(`rateLimitCheck: [RESPONSE] Status: ${res.status}`);

        if (res.status === 429) {
            r.log(`rateLimitCheck: [BLOCKED] Sending 429 to client`);
            r.return(429, "Rate limit exceeded\n");
            return;
        }

        if (res.status !== 200) {
            r.log(`rateLimitCheck: [WARN] Service returned ${res.status}, not 200. Proceeding anyway.`);
        } else {
            r.log(`rateLimitCheck: [SUCCESS] Service returned 200. Proceeding.`);
        }

    } catch (e) {
        r.log(`rate-limit check failed: ${e}`);
    }
    r.return(204);
}

export default {
    rateLimitCheck
};