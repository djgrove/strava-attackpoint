const BASE_URL = "https://www.attackpoint.org";

export class APClient {
  constructor() {
    this.cookies = "";
    this.userId = "";
  }

  async login(username, password) {
    const params = new URLSearchParams({ username, password, returl: "/" });
    const resp = await fetch(`${BASE_URL}/dologin.jsp`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: params.toString(),
      redirect: "manual",
    });

    // Capture cookies from Set-Cookie headers.
    // Use getSetCookie() if available (Node 20+), fall back to raw header parsing.
    let setCookies = [];
    if (typeof resp.headers.getSetCookie === "function") {
      setCookies = resp.headers.getSetCookie();
    } else {
      const raw = resp.headers.get("set-cookie");
      if (raw) setCookies = raw.split(/,(?=\s*\w+=)/);
    }
    this.cookies = setCookies.map((c) => c.split(";")[0].trim()).join("; ");

    if (!this.cookies) throw new Error("Login failed — no session cookie received");

    const location = resp.headers.get("location") || "";
    if (location.includes("login.jsp")) {
      throw new Error("Login failed — check your username and password");
    }

    // Fetch homepage to extract user ID.
    const homeResp = await this.get("/");
    const homeBody = await homeResp.text();
    const userMatch = homeBody.match(/\/user[_/](\d+)/);
    if (userMatch) this.userId = userMatch[1];
  }

  async get(path) {
    return fetch(`${BASE_URL}${path}`, {
      headers: { Cookie: this.cookies },
    });
  }

  async post(path, formData) {
    const params = new URLSearchParams(formData);
    return fetch(`${BASE_URL}${path}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
        Cookie: this.cookies,
      },
      body: params.toString(),
    });
  }

  async discoverForm() {
    const resp = await this.get("/newtraining.jsp");
    const html = await resp.text();

    // Check if we're redirected to login (not authenticated).
    if (html.includes("Please sign in") || html.includes("login.jsp")) {
      throw new Error("AP session expired — not authenticated");
    }

    return parseTrainingForm(html);
  }

  async submitWorkout(formAction, formData) {
    const resp = await this.post(formAction, formData);
    const body = await resp.text();

    if (resp.status >= 400) {
      const errMatch = body.match(/<pre>([^<]+)<\/pre>/);
      throw new Error(errMatch ? errMatch[1] : `submission failed (${resp.status})`);
    }

    // Check for error in response body even on 200.
    if (body.includes("400 Bad Request") || body.includes("missing required")) {
      const errMatch = body.match(/<pre>([^<]+)<\/pre>/);
      throw new Error(errMatch ? errMatch[1] : "submission failed");
    }
  }

  async scanLogForStravaEntries(since) {
    if (!this.userId) return {};

    const entries = {};
    let current = new Date();
    const start = new Date(since);

    while (current >= start) {
      const dateStr = current.toISOString().split("T")[0];
      const resp = await this.get(`/viewlog.jsp/user_${this.userId}/period-7/enddate-${dateStr}`);
      const html = await resp.text();

      // Find tlactivity divs with strava URLs.
      const activityPattern = /data-sessionid="(\d+)"[\s\S]*?class="descrow[^"]*"[^>]*>([\s\S]*?)<\/div>/g;
      let match;
      while ((match = activityPattern.exec(html)) !== null) {
        const sessionId = match[1];
        const desc = match[2];
        const stravaMatch = desc.match(/strava\.com\/activities\/(\d+)/);
        if (stravaMatch) {
          entries[stravaMatch[1]] = sessionId;
        }
      }

      current.setDate(current.getDate() - 7);
    }

    return entries;
  }

  async deleteSession(sessionId) {
    const editResp = await this.get(`/edittrainingsession.jsp?sessionid=${sessionId}`);
    const editHtml = await editResp.text();

    const csrfMatch = editHtml.match(/csrfToken=([^&'"]+)/);
    if (!csrfMatch) throw new Error(`No CSRF token for session ${sessionId}`);

    let csrfToken;
    try {
      csrfToken = decodeURIComponent(csrfMatch[1]);
    } catch {
      csrfToken = csrfMatch[1];
    }

    const resp = await this.post(`/deltraining.jsp?sessionid=${sessionId}`, {
      csrfToken,
    });
    if (resp.status >= 400) throw new Error(`Delete failed for session ${sessionId}`);
  }
}

function parseTrainingForm(html) {
  let action = "/addtraining.jsp";

  // Find all form actions and pick the training one.
  const formPattern = /<form[^>]*action="([^"]*)"[^>]*>/gi;
  let formMatch;
  while ((formMatch = formPattern.exec(html)) !== null) {
    const formAction = formMatch[1];
    // Get the chunk of HTML from this form tag to the next form or end.
    const startIdx = formMatch.index;
    const nextForm = html.indexOf("<form", startIdx + 1);
    const formChunk = nextForm > 0 ? html.substring(startIdx, nextForm) : html.substring(startIdx);

    if (formChunk.includes("activitytypeid")) {
      action = formAction;
      break;
    }
  }

  // Find the activitytypeid select element more broadly.
  // The HTML may use name=activitytypeid (no quotes) or name="activitytypeid".
  const selectPattern = /(?:name=activitytypeid|name="activitytypeid")[^>]*>[\s\S]*?<\/select>/i;
  const typeSelectMatch = html.match(selectPattern);

  const activityTypes = [];
  if (typeSelectMatch) {
    const optionPattern = /value="([^"]*)"[^>]*>([^<]*)/g;
    let optMatch;
    while ((optMatch = optionPattern.exec(typeSelectMatch[0])) !== null) {
      const label = optMatch[2].trim();
      if (label) {
        activityTypes.push({ value: optMatch[1], label });
      }
    }
  }

  return { action, activityTypes };
}
