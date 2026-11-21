#!/usr/bin/env node
/**
 * scripts/check-env.js — build-time environment variable validator.
 *
 * Run before `react-scripts build` (or any CI build step) to catch missing
 * required variables early instead of discovering them at runtime in production.
 *
 * Usage:
 *   node scripts/check-env.js
 *
 * Add to package.json scripts:
 *   "check-env": "node scripts/check-env.js",
 *   "build": "npm run check-env && react-scripts build"
 */

const path = require("path");
const fs = require("fs");

// Load .env if present (mirrors what react-scripts does at dev time)
const envFile = path.resolve(__dirname, "../.env");
if (fs.existsSync(envFile)) {
  require("dotenv").config({ path: envFile });
}

const REQUIRED = [
  "REACT_APP_API_URL",
];

function isValidUrl(str) {
  try {
    const u = new URL(str);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

const errors = [];

for (const key of REQUIRED) {
  const val = process.env[key];
  if (!val || val.trim() === "") {
    errors.push(`  ✗  ${key} is not set`);
  }
}

const apiUrl = process.env.REACT_APP_API_URL;
if (apiUrl && apiUrl.trim() !== "" && !isValidUrl(apiUrl)) {
  errors.push(`  ✗  REACT_APP_API_URL "${apiUrl}" is not a valid http/https URL`);
}

if (errors.length > 0) {
  console.error("\n[check-env] Build aborted — required environment variables are missing or invalid:\n");
  errors.forEach((e) => console.error(e));
  console.error(
    "\nCopy frontend/.env.example to frontend/.env and set the values before building.\n"
  );
  process.exit(1);
}

console.log("[check-env] All required environment variables are set. ✓\n");
