/**
 * Required environment variables:
 *   REACT_APP_API_URL          - Base URL of the Gpay-Remit backend API
 *                                e.g. http://localhost:8080/api/v1
 *   REACT_APP_STELLAR_NETWORK  - Stellar network: "testnet" or "mainnet"
 *                                (optional — defaults to "testnet")
 *
 * In development, copy frontend/.env.example to frontend/.env and fill in the values.
 * Run `node scripts/check-env.js` (or `npm run check-env`) before building to validate
 * all required variables are present.
 */

const DEV_FALLBACK_URL = "http://localhost:8080/api/v1";
const IS_DEV_OR_TEST =
  process.env.NODE_ENV === "development" || process.env.NODE_ENV === "test";

// All variables that must be set in production. Extend this list as new
// required vars are added so check-env.js and runtime validation stay in sync.
export const REQUIRED_ENV_VARS = [
  "REACT_APP_API_URL",
];

export const OPTIONAL_ENV_VARS = [
  "REACT_APP_STELLAR_NETWORK",
];

function isValidUrl(str) {
  try {
    const u = new URL(str);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

/**
 * Validate all required environment variables.
 * In production, throws on the first missing or invalid variable.
 * In development/test, logs a warning and continues.
 *
 * @returns {string[]} List of validation error messages (empty if all pass)
 */
export function validateEnvVars() {
  const errors = [];

  for (const key of REQUIRED_ENV_VARS) {
    const val = process.env[key];
    if (!val || val.trim() === "") {
      errors.push(`${key} is required but not set.`);
    }
  }

  // Extra semantic check for the API URL
  const rawUrl = process.env.REACT_APP_API_URL;
  if (rawUrl && rawUrl.trim() !== "" && !isValidUrl(rawUrl)) {
    errors.push(
      `REACT_APP_API_URL "${rawUrl}" is not a valid http/https URL. ` +
        "Provide a full URL including protocol, e.g. https://api.example.com/api/v1"
    );
  }

  if (errors.length > 0 && !IS_DEV_OR_TEST) {
    throw new Error(
      "[Gpay-Remit] Missing or invalid environment variables:\n" +
        errors.map((e) => `  • ${e}`).join("\n") +
        "\nSet them in your environment or .env file before building."
    );
  }

  if (errors.length > 0 && IS_DEV_OR_TEST) {
    console.warn(
      "[Gpay-Remit] Environment variable warnings (using fallbacks in dev/test):\n" +
        errors.map((e) => `  • ${e}`).join("\n")
    );
  }

  return errors;
}

function resolveApiUrl() {
  const raw = process.env.REACT_APP_API_URL;

  if (!raw || raw.trim() === "") {
    if (IS_DEV_OR_TEST) {
      console.warn(
        `[Gpay-Remit] REACT_APP_API_URL is not set. Using development fallback: ${DEV_FALLBACK_URL}`
      );
      return DEV_FALLBACK_URL;
    }
    throw new Error(
      "[Gpay-Remit] REACT_APP_API_URL is required but not set. " +
        "Set it in your environment or .env file before starting the app."
    );
  }

  if (!isValidUrl(raw)) {
    if (IS_DEV_OR_TEST) {
      console.warn(
        `[Gpay-Remit] REACT_APP_API_URL "${raw}" is not a valid http/https URL. ` +
          `Using development fallback: ${DEV_FALLBACK_URL}`
      );
      return DEV_FALLBACK_URL;
    }
    throw new Error(
      `[Gpay-Remit] REACT_APP_API_URL "${raw}" is not a valid http/https URL. ` +
        "Provide a full URL including protocol, e.g. https://api.example.com/api/v1"
    );
  }

  return raw.replace(/\/$/, "");
}

// Validate on module load so misconfigured builds surface the error immediately.
validateEnvVars();

export const API_BASE_URL = resolveApiUrl();

export const STELLAR_NETWORK =
  process.env.REACT_APP_STELLAR_NETWORK || "testnet";
