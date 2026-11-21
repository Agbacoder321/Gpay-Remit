// Hook that fetches and caches exchange rates, with retry support.
import { useState, useEffect, useRef, useCallback } from "react";
import { getExchangeRate } from "../services/api";

const RETRY_DELAYS_MS = [2000, 5000, 10000]; // exponential-ish back-off

/**
 * @param {string} fromCurrency
 * @param {string} toCurrency
 * @param {object} [opts]
 * @param {number} [opts.refreshIntervalMs=60000] - How often to re-fetch the rate (0 = no polling)
 * @returns {{ rate: number|null, lastKnownRate: number|null, loading: boolean, error: string|null, retry: function }}
 */
export function useExchangeRate(fromCurrency, toCurrency, { refreshIntervalMs = 60_000 } = {}) {
  const [rate, setRate] = useState(null);
  const [lastKnownRate, setLastKnownRate] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const retryCountRef = useRef(0);
  const retryTimerRef = useRef(null);
  const pollTimerRef = useRef(null);
  const mountedRef = useRef(true);

  const fetchRate = useCallback(
    async (isRetry = false) => {
      if (!fromCurrency || !toCurrency || fromCurrency === toCurrency) return;

      clearTimeout(retryTimerRef.current);

      if (!isRetry) {
        retryCountRef.current = 0;
      }

      setLoading(true);
      setError(null);

      try {
        const res = await getExchangeRate(fromCurrency, toCurrency);
        const fetched = res?.data?.rate ?? res?.data;
        const numeric = typeof fetched === "number" ? fetched : parseFloat(fetched);

        if (!mountedRef.current) return;

        if (!isNaN(numeric) && numeric > 0) {
          setRate(numeric);
          setLastKnownRate(numeric);
          retryCountRef.current = 0;
        } else {
          throw new Error("Invalid rate received from server.");
        }
      } catch (err) {
        if (!mountedRef.current) return;

        const msg =
          err?.response?.data?.error ||
          err?.message ||
          "Exchange rate unavailable.";
        setRate(null);
        setError(msg);

        // Schedule an automatic retry if we have attempts left
        const attempt = retryCountRef.current;
        if (attempt < RETRY_DELAYS_MS.length) {
          retryCountRef.current += 1;
          retryTimerRef.current = setTimeout(() => {
            fetchRate(true);
          }, RETRY_DELAYS_MS[attempt]);
        }
      } finally {
        if (mountedRef.current) setLoading(false);
      }
    },
    [fromCurrency, toCurrency]
  );

  // Manual retry resets the auto-retry counter and fetches immediately.
  const retry = useCallback(() => {
    clearTimeout(retryTimerRef.current);
    retryCountRef.current = 0;
    fetchRate(false);
  }, [fetchRate]);

  useEffect(() => {
    mountedRef.current = true;
    fetchRate(false);

    if (refreshIntervalMs > 0) {
      pollTimerRef.current = setInterval(() => fetchRate(false), refreshIntervalMs);
    }

    return () => {
      mountedRef.current = false;
      clearTimeout(retryTimerRef.current);
      clearInterval(pollTimerRef.current);
    };
  }, [fetchRate, refreshIntervalMs]);

  return { rate, lastKnownRate, loading, error, retry };
}
