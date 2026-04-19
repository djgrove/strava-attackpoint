import { fetchActivities, fetchActivityDetail, estimateIntensity, verifyAthleteToken } from "./strava.mjs";
import { APClient } from "./attackpoint.mjs";
import { mapActivity } from "./mapping.mjs";
import { getUser, putUser, deleteUser, updateTokens, updateLastSync, updateStatus } from "./dynamo.mjs";
import { SQSClient, SendMessageCommand } from "@aws-sdk/client-sqs";

const STRAVA_TOKEN_URL = "https://www.strava.com/oauth/token";
const ALLOWED_ORIGIN = "*";

const sqsClient = new SQSClient({});

export const handler = async (event) => {
  // SQS trigger — process queued webhook events.
  if (event.Records) {
    for (const record of event.Records) {
      const webhookEvent = JSON.parse(record.body);
      await processWebhookSync(webhookEvent);
    }
    return;
  }

  const method = event.requestContext?.http?.method;
  const path = event.rawPath;

  // CORS preflight.
  if (method === "OPTIONS") {
    return corsResponse(204, "");
  }

  // Strava webhook validation (GET).
  if (method === "GET" && path === "/webhook") {
    return handleWebhookValidation(event);
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
    if (path === "/webhook") return await handleWebhookEvent(body);
    if (path === "/register") return corsResponse(200, await handleRegister(body));
    if (path === "/unregister") return corsResponse(200, await handleUnregister(body));
    if (path === "/autosync-status") return corsResponse(200, await handleAutoSyncStatus(body));
    return corsResponse(404, { error: "Not found" });
  } catch (err) {
    return corsResponse(500, { error: err.message });
  }
};

// --- OAuth endpoints (unchanged) ---

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

// --- Manual sync (unchanged logic, refactored to use syncSingleActivity) ---

async function handleSync(body) {
  const { strava_access_token, ap_username, ap_password, since, end } = body;
  if (!strava_access_token) throw new Error("Missing strava_access_token");
  if (!ap_username || !ap_password) throw new Error("Missing AP credentials");
  if (!since) throw new Error("Missing since date");

  const ap = new APClient();
  await ap.login(ap_username, ap_password);

  const form = await ap.discoverForm();
  if (form.activityTypes.length === 0) {
    throw new Error("No activity types found in AP form — AP login may have failed silently");
  }

  const activities = await fetchActivities(strava_access_token, since, end);
  if (activities.length === 0) {
    return { results: [], summary: { synced: 0, skipped: 0, failed: 0 } };
  }

  const existing = await ap.scanLogForStravaEntries(since);

  const results = [];
  for (const activity of activities) {
    const result = await syncSingleActivity(strava_access_token, activity, ap, form, existing);
    results.push(result);
  }

  const summary = {
    synced: results.filter((r) => r.status === "synced").length,
    skipped: results.filter((r) => r.status === "skipped").length,
    failed: results.filter((r) => r.status === "failed").length,
  };

  return { results, summary };
}

// --- Shared sync logic ---

async function syncSingleActivity(accessToken, activity, ap, form, existing) {
  const id = String(activity.id);
  const result = { name: activity.name, strava_id: activity.id, status: "synced" };

  try {
    const detail = await fetchActivityDetail(accessToken, activity.id);
    const merged = { ...activity, ...detail };

    const intensity = merged.has_heartrate
      ? estimateIntensity(merged.average_heartrate, merged.max_heartrate)
      : 0;

    const mapped = mapActivity(merged, form.activityTypes, intensity);

    if (!mapped.formData.activitytypeid) {
      throw new Error("No matching activity type in AP account");
    }

    // Delete existing only after validating replacement.
    if (existing && existing[id]) {
      await ap.deleteSession(existing[id]);
    }

    await ap.submitWorkout(form.action, mapped.formData);

    if (mapped.warning) result.warning = mapped.warning;
  } catch (err) {
    result.status = "failed";
    result.error = err.message;
  }

  return result;
}

// --- Webhook endpoints ---

function handleWebhookValidation(event) {
  const params = event.queryStringParameters || {};
  const mode = params["hub.mode"];
  const challenge = params["hub.challenge"];
  const verifyToken = params["hub.verify_token"];

  if (mode === "subscribe" && verifyToken === process.env.STRAVA_WEBHOOK_VERIFY_TOKEN) {
    return {
      statusCode: 200,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ "hub.challenge": challenge }),
    };
  }

  return { statusCode: 403, body: "Verification failed" };
}

async function handleWebhookEvent(body) {
  const { object_type, object_id, aspect_type, owner_id } = body;

  // Only process new or updated activities.
  if (object_type !== "activity" || (aspect_type !== "create" && aspect_type !== "update")) {
    return { statusCode: 200, body: "OK" };
  }

  // Queue for async processing.
  await sqsClient.send(
    new SendMessageCommand({
      QueueUrl: process.env.SQS_QUEUE_URL,
      MessageBody: JSON.stringify({ object_id, owner_id, aspect_type }),
    })
  );

  return { statusCode: 200, body: "OK" };
}

// --- Webhook sync processor (from SQS) ---

async function processWebhookSync({ object_id, owner_id }) {
  const user = await getUser(owner_id);
  if (!user || user.status !== "active") return;

  try {
    // Refresh Strava token if needed.
    let accessToken = user.strava_access_token;
    const now = Math.floor(Date.now() / 1000);
    if (now >= (user.strava_token_expires_at || 0)) {
      const refreshed = await forwardToStrava({
        client_id: process.env.STRAVA_CLIENT_ID,
        client_secret: process.env.STRAVA_CLIENT_SECRET,
        refresh_token: user.strava_refresh_token,
        grant_type: "refresh_token",
      });
      if (refreshed.access_token) {
        accessToken = refreshed.access_token;
        await updateTokens(owner_id, refreshed.access_token, refreshed.refresh_token, refreshed.expires_at);
      } else {
        await updateStatus(owner_id, "strava_disconnected");
        return;
      }
    }

    // Fetch the specific activity.
    const activity = await fetchActivityDetail(accessToken, object_id);
    if (!activity) return;

    // Login to AP.
    const ap = new APClient();
    await ap.login(user.ap_username, user.ap_password);

    const form = await ap.discoverForm();
    if (form.activityTypes.length === 0) {
      await updateStatus(owner_id, "ap_error");
      return;
    }

    // Check for existing entry.
    const startDate = new Date(activity.start_date_local);
    const since = new Date(startDate);
    since.setDate(since.getDate() - 1);
    const existing = await ap.scanLogForStravaEntries(since.toISOString().split("T")[0]);

    await syncSingleActivity(accessToken, activity, ap, form, existing);
    await updateLastSync(owner_id);
  } catch (err) {
    if (err.message.includes("Login failed") || err.message.includes("check your username")) {
      await updateStatus(owner_id, "ap_credentials_invalid");
    } else if (err.message.includes("Strava token expired") || err.message.includes("Invalid Strava token")) {
      await updateStatus(owner_id, "strava_disconnected");
    }
    // Let SQS retry on other errors (up to 3 times, then DLQ).
    throw err;
  }
}

// --- Registration endpoints ---

async function handleRegister(body) {
  const { strava_access_token, strava_refresh_token, strava_token_expires_at, ap_username, ap_password } = body;

  if (!strava_access_token || !strava_refresh_token) {
    throw new Error("Missing Strava tokens");
  }
  if (!ap_username || !ap_password) {
    throw new Error("Missing AP credentials");
  }

  // Verify Strava token and get athlete ID.
  const athleteId = await verifyAthleteToken(strava_access_token);

  // Validate AP credentials by attempting login.
  const ap = new APClient();
  await ap.login(ap_username, ap_password);

  // Store user.
  await putUser({
    strava_athlete_id: athleteId,
    ap_username,
    ap_password,
    strava_access_token,
    strava_refresh_token,
    strava_token_expires_at: strava_token_expires_at || 0,
  });

  return { success: true, athlete_id: athleteId };
}

async function handleUnregister(body) {
  const { strava_access_token } = body;
  if (!strava_access_token) throw new Error("Missing Strava token");

  const athleteId = await verifyAthleteToken(strava_access_token);
  await deleteUser(athleteId);

  return { success: true };
}

async function handleAutoSyncStatus(body) {
  const { strava_access_token } = body;
  if (!strava_access_token) throw new Error("Missing Strava token");

  const athleteId = await verifyAthleteToken(strava_access_token);
  const user = await getUser(athleteId);

  if (!user) {
    return { registered: false };
  }

  return {
    registered: true,
    status: user.status,
    last_sync_at: user.last_sync_at || null,
  };
}

// --- Helpers ---

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
      "Access-Control-Allow-Methods": "POST, GET, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type",
    },
    body: typeof body === "string" ? body : JSON.stringify(body),
  };
}
