const STRAVA_TOKEN_URL = "https://www.strava.com/oauth/token";

export const handler = async (event) => {
  const method = event.requestContext?.http?.method;
  const path = event.rawPath;

  if (method !== "POST") {
    return response(405, { error: "Method not allowed" });
  }

  let body;
  try {
    body = JSON.parse(event.body || "{}");
  } catch {
    return response(400, { error: "Invalid JSON body" });
  }

  const clientId = process.env.STRAVA_CLIENT_ID;
  const clientSecret = process.env.STRAVA_CLIENT_SECRET;

  if (!clientId || !clientSecret) {
    return response(500, { error: "Server misconfigured" });
  }

  if (path === "/token") {
    return handleTokenExchange(body, clientId, clientSecret);
  } else if (path === "/refresh") {
    return handleTokenRefresh(body, clientId, clientSecret);
  }

  return response(404, { error: "Not found" });
};

async function handleTokenExchange(body, clientId, clientSecret) {
  const { code } = body;
  if (!code) {
    return response(400, { error: "Missing required field: code" });
  }

  return forwardToStrava({
    client_id: clientId,
    client_secret: clientSecret,
    code,
    grant_type: "authorization_code",
  });
}

async function handleTokenRefresh(body, clientId, clientSecret) {
  const { refresh_token } = body;
  if (!refresh_token) {
    return response(400, { error: "Missing required field: refresh_token" });
  }

  return forwardToStrava({
    client_id: clientId,
    client_secret: clientSecret,
    refresh_token,
    grant_type: "refresh_token",
  });
}

async function forwardToStrava(params) {
  const resp = await fetch(STRAVA_TOKEN_URL, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams(params).toString(),
  });

  const data = await resp.json();
  return response(resp.status, data);
}

function response(statusCode, body) {
  return {
    statusCode,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}
