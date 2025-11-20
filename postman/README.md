# ExLink Postman API Collections

This directory contains Postman collections and environments for testing and debugging exchange APIs (Binance, Bybit, Gate, and OKX).

## Quick Start

### 1. Import Collections

1. Open Postman application
2. Click **Import** button (top left)
3. Select **File** tab
4. Navigate to `postman/collections/` directory
5. Select the collections you want to import:
   - `exlink-binance.collection.json`
   - `exlink-bybit.collection.json`
   - `exlink-gate.collection.json`
   - `exlink-okx.collection.json`
6. Click **Import**

### 2. Import Environments

1. Click **Import** button in Postman
2. Select **File** tab
3. Navigate to `postman/environments/` directory
4. Import both environment files:
   - `exlink-mainnet.environment.json` - Production environment (real trading)
   - `exlink-testnet.environment.json` - Test/sandbox environment (simulated trading)
5. Click **Import**

### 3. Configure API Keys

1. In Postman, click the **Environments** icon (top right)
2. Select the environment you want to configure (e.g., "ExLink - Testnet")
3. Fill in the required variables for each exchange:

**Binance:**
   - `binance_api_key` - Your Binance API Key
   - `binance_secret_key` - Your Binance Secret Key

**Bybit:**
   - `bybit_api_key` - Your Bybit API Key
   - `bybit_secret_key` - Your Bybit Secret Key

**Gate:**
   - `gate_api_key` - Your Gate API Key
   - `gate_secret_key` - Your Gate Secret Key

**OKX:**
   - `okx_api_key` - Your OKX API Key
   - `okx_secret_key` - Your OKX Secret Key
   - `okx_passphrase` - Your OKX Passphrase

4. Click **Save** to save your changes

### 4. Select Environment

1. In Postman, click the environment dropdown (top right)
2. Select the environment you want to use:
   - **ExLink - Mainnet** - For production trading
   - **ExLink - Testnet** - For testing and development
3. Now all requests will use variables from the selected environment

## Collections Overview

Each exchange has a unified API collection that contains all trading and account management APIs:

- **exlink-binance.collection.json** - Binance API collection (trading and account APIs)
- **exlink-bybit.collection.json** - Bybit API collection (trading and account APIs)
- **exlink-gate.collection.json** - Gate API collection (trading and account APIs)
- **exlink-okx.collection.json** - OKX API collection (trading and account APIs)

Each collection includes:

**Trading APIs (6 endpoints):**
- Spot - Buy
- Spot - Sell
- Swap - Open Long (Buy to Open Long)
- Swap - Close Long (Sell to Close Long)
- Swap - Open Short (Sell to Open Short)
- Swap - Close Short (Buy to Close Short)

**Account APIs (2 endpoints):**
- Spot - Get Account Balance
- Swap - Get Account Positions

All collections have authentication scripts integrated at the collection level, so you don't need to manually add scripts. The Pre-request Scripts will automatically generate signatures for all requests.

## Environments

There are two unified environment files that contain variables for all exchanges:

- **exlink-mainnet.environment.json** - Production environment (real trading)
- **exlink-testnet.environment.json** - Test/sandbox environment (simulated trading)

### Environment Variables

Each environment file contains variables for all exchanges:

**Binance:**
- `binance_base_url` - API base URL (automatically set based on environment)
- `binance_fapi_base_url` - Futures API base URL (automatically set based on environment)
- `binance_api_key` - Your API key
- `binance_secret_key` - Your secret key

**Bybit:**
- `bybit_base_url` - API base URL
- `bybit_api_key` - Your API key
- `bybit_secret_key` - Your secret key

**Gate:**
- `gate_base_url` - API base URL
- `gate_api_key` - Your API key
- `gate_secret_key` - Your secret key

**OKX:**
- `okx_base_url` - API base URL
- `okx_api_key` - Your API key
- `okx_secret_key` - Your secret key
- `okx_passphrase` - Your passphrase (set when creating API key)
- `okx_testnet` - "true" for testnet, "false" for mainnet

## Authentication

All collections have authentication scripts integrated at the collection level. The Pre-request Scripts automatically:

- Generate timestamps
- Build query strings and request bodies
- Create HMAC signatures
- Set required authentication headers

**No manual script configuration is needed** - authentication is handled automatically for all requests in each collection.

### Signature Methods

- **Binance**: HMAC-SHA256 signature with query parameters
- **Bybit**: HMAC-SHA256 signature (v5 API format)
- **Gate**: HMAC-SHA512 signature
- **OKX**: HMAC-SHA256 signature (Base64 encoded)

## Usage Examples

### Example 1: Query Account Balance

1. Ensure you've configured API keys in the environment
2. Select the appropriate API collection (e.g., "ExLink - Binance")
3. Select the correct environment (Mainnet or Testnet)
4. Choose **Spot - Get Account Balance** request
5. Click **Send**
6. The Pre-request Script will automatically generate the signature

### Example 2: Get Contract Positions

1. Select the appropriate API collection
2. Select the correct environment with API keys configured
3. Choose **Swap - Get Account Positions** request
4. Optionally modify the query parameters (e.g., symbol filter)
5. Click **Send**

### Example 3: Create a Spot Order

1. Select the appropriate API collection
2. Select the correct environment with API keys configured
3. Choose **Spot - Buy** or **Spot - Sell** request
4. Modify the request body parameters (instId/symbol, sz/quantity, etc.)
5. Click **Send**

### Example 4: Open a Long Position

1. Select the appropriate API collection
2. Select the correct environment with API keys configured
3. Choose **Swap - Open Long (Buy to Open Long)** request
4. Modify the request body parameters (instId/symbol, sz/quantity, etc.)
5. Click **Send**

## API Endpoints

All collections follow the same structure with 8 endpoints per exchange:

### Trading APIs (6 endpoints)

1. **Spot - Buy** - Buy spot assets
2. **Spot - Sell** - Sell spot assets
3. **Swap - Open Long (Buy to Open Long)** - Open a long position in perpetual swap
4. **Swap - Close Long (Sell to Close Long)** - Close a long position in perpetual swap
5. **Swap - Open Short (Sell to Open Short)** - Open a short position in perpetual swap
6. **Swap - Close Short (Buy to Close Short)** - Close a short position in perpetual swap

### Account APIs (2 endpoints)

1. **Spot - Get Account Balance** - Get account balance for all currencies
2. **Swap - Get Account Positions** - Get contract positions (optional symbol filter)

### Exchange-Specific Details

**Binance:**
- Trading APIs use `/fapi/v1/order` for swaps and `/api/v3/order` for spot
- Uses query parameters (not JSON body) for requests
- Requires `reduceOnly=true` parameter to close positions
- Separate base URLs for spot (`binance_base_url`) and futures (`binance_fapi_base_url`)

**Bybit:**
- All APIs use `/v5/order` endpoint
- Uses JSON body for requests
- Requires `reduceOnly=false` to open positions, `reduceOnly=true` to close positions

**Gate:**
- Swap APIs use `/api/v4/futures/usdt/orders`
- Spot APIs use `/api/v4/spot/orders`
- Uses `size` parameter (positive for buy, negative for sell) for swap orders

**OKX:**
- All APIs use `/api/v5/trade/order` for trading and `/api/v5/account/*` for account
- Uses `posSide` parameter (long/short) to specify position direction
- Uses `tdMode` parameter (cross/isolated/cash) for margin mode

## Security Best Practices

1. **Never commit API keys** - Keep your API keys secure and never commit them to version control
2. **Use testnet first** - Always test in testnet/sandbox environment before using mainnet
3. **Minimal permissions** - Set minimum required permissions for your API keys (read-only for testing)
4. **IP whitelist** - Configure IP whitelist in exchange settings if possible
5. **Regular rotation** - Rotate your API keys regularly

## Troubleshooting

### Signature Errors

If you encounter signature errors:

1. Verify API keys are correctly set in the environment
2. Check that the correct environment is selected
3. Ensure timestamps are correct (usually handled automatically by scripts)
4. Verify query parameters and request body format

### Environment Not Working

1. Make sure you've imported the environment file
2. Verify the environment is selected in the dropdown
3. Check that all required variables are set
4. Ensure variable names match exactly (case-sensitive)

### Request Fails

1. Check the API endpoint URL is correct
2. Verify you're using the correct environment (testnet vs mainnet)
3. Check request parameters are valid
4. Review error response for details

## Reference Documentation

- [Binance Spot API Documentation](https://developers.binance.com/docs/binance-spot-api-docs/rest-api/general-api-information)
- [Binance USDⓈ-M API Documentation](https://developers.binance.com/docs/derivatives/usds-margined-futures/general-info)
- [Binance COIN-M API Documentation](https://developers.binance.com/docs/derivatives/coin-margined-futures/general-info)
- [Bybit API Documentation](https://bybit-exchange.github.io/docs/v5/intro)
- [Gate API Documentation](https://www.gate.com/docs/developers/apiv4/)
- [OKX API Documentation](https://www.okx.com/docs-v5/en/)

## Notes

- All collections use environment variables for base URLs, so switching between testnet and mainnet is as simple as changing the environment
- **Authentication scripts are integrated at the collection level** - No manual script configuration is needed
- All API collections require authentication (API keys) which should be configured in the environment
- Collections are unified and contain both trading and account management APIs
- Each collection includes 8 endpoints: 6 trading APIs (swap open/close long/short, spot buy/sell) and 2 account APIs (balance, positions)
- All exchanges share the same two environment files (mainnet and testnet), making it easy to switch between environments
