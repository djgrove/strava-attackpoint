const STRAVA_TO_AP_KEYWORDS = {
  Run: ["run"],
  TrailRun: ["run"],
  VirtualRun: ["run"],
  Ride: ["bike", "cycl"],
  MountainBikeRide: ["bike", "cycl", "mtb"],
  GravelRide: ["bike", "cycl"],
  EBikeRide: ["bike", "cycl"],
  VirtualRide: ["bike", "cycl"],
  Swim: ["swim"],
  NordicSki: ["ski"],
  BackcountrySki: ["ski"],
  RollerSki: ["ski"],
  Rowing: ["row"],
  Canoeing: ["paddl", "canoe", "kayak"],
  Kayaking: ["paddl", "kayak", "canoe"],
  StandUpPaddling: ["paddl"],
  Hike: ["hik"],
  Walk: ["walk", "hik"],
  WeightTraining: ["weight", "strength"],
  Crossfit: ["cross-training", "crossfit"],
  Yoga: ["stretch", "yoga"],
  Workout: ["cross-training", "core"],
};

export function mapActivityType(sportType, name, description, apTypes) {
  const validTypes = apTypes.filter((t) => t.value !== "-1");

  // Orienteering override.
  const nameLower = (name || "").toLowerCase();
  const descLower = (description || "").toLowerCase();
  if (nameLower.includes("orienteering") || descLower.includes("orienteering")) {
    const match = findByKeyword(validTypes, "orient");
    if (match) return { id: match.value, name: match.label, warning: "" };
  }

  // Keyword matching.
  const keywords = STRAVA_TO_AP_KEYWORDS[sportType];
  if (keywords) {
    for (const kw of keywords) {
      const match = findByKeyword(validTypes, kw);
      if (match) return { id: match.value, name: match.label, warning: "" };
    }
  }

  // Fallback.
  if (validTypes.length > 0) {
    const warning = `no AP type matching '${sportType}' — using '${validTypes[0].label}'`;
    return { id: validTypes[0].value, name: validTypes[0].label, warning };
  }

  return { id: "", name: "", warning: "no activity types found" };
}

function findByKeyword(types, keyword) {
  const kw = keyword.toLowerCase();
  return types.find((t) => t.label.toLowerCase().includes(kw));
}

export function mapActivity(activity, apTypes, intensity) {
  const type = mapActivityType(
    activity.sport_type,
    activity.name,
    activity.description,
    apTypes
  );

  const startDate = new Date(activity.start_date_local);

  return {
    formData: {
      activitytypeid: type.id,
      "session-day": String(startDate.getDate()).padStart(2, "0"),
      "session-month": String(startDate.getMonth() + 1).padStart(2, "0"),
      "session-year": String(startDate.getFullYear()),
      sessionstarthour: String(startDate.getHours()),
      distance: activity.distance > 0 ? (activity.distance / 1609.344).toFixed(2) : "",
      distanceunits: "miles",
      sessionlength: formatDuration(activity.moving_time),
      ahr: activity.average_heartrate ? Math.round(activity.average_heartrate).toString() : "",
      mhr: activity.max_heartrate ? Math.round(activity.max_heartrate).toString() : "",
      climb: activity.total_elevation_gain > 0 ? Math.round(activity.total_elevation_gain).toString() : "",
      intensity: String(intensity),
      description: buildDescription(activity),
      isplan: "0",
      workouttypeid: "1",
      map: "0",
      shoes: "null",
      restday: "",
      sick: "",
      injured: "",
      spiked: "",
      controls: "",
      weight: "",
      rhr: "",
      sleep: "",
      pace: "",
      wunit: "",
      climb_grade: "",
      climb_angle: "",
      newactivitytype: "",
      activitymodifiers: "",
    },
    warning: type.warning,
  };
}

function formatDuration(seconds) {
  if (!seconds || seconds <= 0) return "";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}${String(m).padStart(2, "0")}${String(s).padStart(2, "0")}`;
  return `${m}${String(s).padStart(2, "0")}`;
}

function buildDescription(activity) {
  const parts = [];
  if (activity.description) parts.push(activity.description);
  parts.push(`https://www.strava.com/activities/${activity.id}`);
  return parts.join("\n\n");
}
