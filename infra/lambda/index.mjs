import { fetchActivities, fetchActivityDetail, estimateIntensity } from "./strava.mjs";
import { APClient } from "./attackpoint.mjs";
import { mapActivity } from "./mapping.mjs";

const STRAVA_TOKEN_URL = "https://www.strava.com/oauth/token";
const ALLOWED_ORIGIN = "*";

export const handler = async (event) => {
  const method = event.requestContext?.http?.method;
  const path = event.rawPath;

  // Handle CORS preflight.
  if (method === "OPTIONS") {
    return corsResponse(204, "");
  }

  if (method !== "POST") {
    return corsResponse(405, { error: "Method not allowed" });
  }

  let body;
  try {
    body = JSON.parse(event.body || "{}");
  } catch {
    return corsResponse(400, { error: "Invalid JSON body" });
  }

  try {
    if (path === "/token") return corsResponse(200, await handleTokenExchange(body));
    if (path === "/refresh") return corsResponse(200, await handleTokenRefresh(body));
    if (path === "/sync") return corsResponse(200, await handleSync(body));
    return corsResponse(404, { error: "Not found" });
  } catch (err) {
    return corsResponse(500, { error: err.message });
  }
};

async function handleTokenExchange(body) {
  const { code } = body;
  if (!code) throw new Error("Missing required field: code");

  return forwardToStrava({
    client_id: process.env.STRAVA_CLIENT_ID,
    client_secret: process.env.STRAVA_CLIENT_SECRET,
    code,
    grant_type: "authorization_code",
  });
}

async function handleTokenRefresh(body) {
  const { refresh_token } = body;
  if (!refresh_token) throw new Error("Missing required field: refresh_token");

  return forwardToStrava({
    client_id: process.env.STRAVA_CLIENT_ID,
    client_secret: process.env.STRAVA_CLIENT_SECRET,
    refresh_token,
    grant_type: "refresh_token",
  });
}

async function handleSync(body) {
  const { strava_access_token, ap_username, ap_password, since, end } = body;

  if (!strava_access_token) throw new Error("Missing strava_access_token");
  if (!ap_username || !ap_password) throw new Error("Missing AP credentials");
  if (!since) throw new Error("Missing since date");

  // 1. Login to AP.
  const ap = new APClient();
  await ap.login(ap_username, ap_password);

  // 2. Discover form.
  const form = await ap.discoverForm();
  if (form.activityTypes.length === 0) {
    throw new Error("No activity types found in AP form");
  }

  // 3. Fetch Strava activities.
  const activities = await fetchActivities(strava_access_token, since, end);
  if (activities.length === 0) {
    return { results: [], summary: { synced: 0, skipped: 0, failed: 0 } };
  }

  // 4. Scan AP log for existing entries.
  const existing = await ap.scanLogForStravaEntries(since);

  // 5. Sync each activity.
  const results = [];
  for (const activity of activities) {
    const id = String(activity.id);
    const result = { name: activity.name, strava_id: activity.id, status: "synced" };

    try {
      // Check if already on AP.
      if (existing[id]) {
        // Delete and replace.
        await ap.deleteSession(existing[id]);
      }

      // Fetch full details.
      const detail = await fetchActivityDetail(strava_access_token, activity.id);
      const merged = { ...activity, ...detail };

      // Estimate intensity.
      const intensity = merged.has_heartrate
        ? estimateIntensity(merged.average_heartrate, merged.max_heartrate)
        : 0;

      // Map and submit.
      const mapped = mapActivity(merged, form.activityTypes, intensity);
      await ap.submitWorkout(form.action, mapped.formData);

      if (mapped.warning) result.warning = mapped.warning;
    } catch (err) {
      result.status = "failed";
      result.error = err.message;
    }

    results.push(result);
  }

  const summary = {
    synced: results.filter((r) => r.status === "synced").length,
    skipped: results.filter((r) => r.status === "skipped").length,
    failed: results.filter((r) => r.status === "failed").length,
  };

  return { results, summary };
}

async function forwardToStrava(params) {
  const resp = await fetch(STRAVA_TOKEN_URL, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams(params).toString(),
  });
  return resp.json();
}

function corsResponse(statusCode, body) {
  return {
    statusCode,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": ALLOWED_ORIGIN,
      "Access-Control-Allow-Methods": "POST, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type",
    },
    body: typeof body === "string" ? body : JSON.stringify(body),
  };
}
