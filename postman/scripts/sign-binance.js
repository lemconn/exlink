// Binance signature script
// Usage: Include this script in Pre-request Script tab
// Requires: binance_api_key, binance_secret_key environment variables

// Generate current timestamp in milliseconds
const timestamp = Date.now();

// Get existing query parameters
const queryParams = pm.request.url.query.toObject();

// Build parameters object (start with timestamp)
const params = {
    timestamp: timestamp
};

// Add existing query parameters (if any)
Object.keys(queryParams).forEach(key => {
    if (key !== 'timestamp' && key !== 'signature') {
        params[key] = queryParams[key];
    }
});

// Build query string for signature (sort keys alphabetically)
const queryString = Object.keys(params)
    .sort()
    .map(key => `${key}=${encodeURIComponent(params[key])}`)
    .join('&');

// HMAC-SHA256 signature
const signature = CryptoJS.HmacSHA256(queryString, pm.environment.get('binance_secret_key')).toString();

// Add signature to parameters
params.signature = signature;

// Update URL query parameters using upsert (more reliable than add)
pm.request.url.query.clear();
Object.keys(params).forEach(key => {
    pm.request.url.query.upsert({
        key: key,
        value: String(params[key])  // Ensure value is string
    });
});

// Set API key header
pm.request.headers.upsert({
    key: 'X-MBX-APIKEY',
    value: pm.environment.get('binance_api_key')
});

