/**
 * Formats monetary amounts from minor units (cents / stroops).
 * @param {number|string} amount Amount in minor units (cents for fiat, stroops for XLM)
 * @param {string} [currency='USD'] Currency code
 * @returns {string} Formatted decimal string
 */
export function formatMinorUnits(amount, currency = 'USD') {
  if (amount == null || isNaN(amount)) return '0.00';
  const val = Number(amount);
  if (currency && (currency.toUpperCase() === 'XLM' || currency.toUpperCase() === 'NATIVE')) {
    return (val / 10000000).toFixed(7);
  }
  return (val / 100).toFixed(2);
}
