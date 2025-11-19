# ExLink Postman API Collections

This directory contains Postman collections, environments, and scripts for testing and debugging exchange APIs (Binance, Bybit, Gate, and OKX).

## Quick Start

### 1. Import Collections

1. Open Postman application
2. Click **Import** button (top left)
3. Select **File** tab
4. Navigate to `postman/collections/` directory
5. Select the collections you want to import (e.g., `binance/exlink-binance-market.collection.json`)
6. Click **Import**

### 2. Import Environments

1. Click **Import** button in Postman
2. Select **File** tab
3. Navigate to `postman/environments/` directory
4. Select the environment files you need (e.g., `exlink-binance-testnet.environment.json`)
5. Click **Import**

### 3. Configure API Keys

1. In Postman, click the **Environments** icon (top right)
2. Select the environment you want to configure (e.g., "ExLink - Binance Testnet")
3. Fill in the required variables:
   - `binance_api_key` - Your Binance API Key
   - `binance_secret_key` - Your Binance Secret Key
   - (Similar for other exchanges)

4. Click **Save** to save your changes

### 4. Select Environment

1. In Postman, click the environment dropdown (top right)
2. Select the environment you want to use (e.g., "ExLink - Binance Testnet")
3. Now all requests will use variables from the selected environment

### 5. Add Authentication Scripts to Collections

For private APIs (Account and Trading collections), you need to add authentication scripts manually. Postman collections don't support external script files, so you need to copy the script content into the collection's Pre-request Script.

**Option A: Add Script to Collection Level (Recommended)**

This applies the script to all requests in the collection:

1. Open the collection in Postman (e.g., "ExLink - Binance Trading")
2. Click on the collection name to open collection settings
3. Go to the **Pre-request Script** tab
4. Open the corresponding script file from `postman/scripts/` directory:
   - For Binance: `postman/scripts/sign-binance.js`
   - For Bybit: `postman/scripts/sign-bybit.js`
   - For Gate: `postman/scripts/sign-gate.js`
   - For OKX: `postman/scripts/sign-okx.js`
5. Copy the entire script content
6. Paste it into the Pre-request Script tab
7. Click **Save**

**Option B: Add Script to Individual Request**

If different requests need different signing logic:

1. Open a specific request in the collection
2. Go to the **Pre-request Script** tab
3. Copy the script content from `postman/scripts/` directory
4. Paste it into the Pre-request Script tab
5. Modify the script if needed for that specific request

**Note:** The scripts in `postman/scripts/` are templates. You need to manually copy their content into Postman's Pre-request Script editor, as Postman doesn't support importing external JavaScript files.

## Collections Overview

### Market Data Collections

Market data collections contain public APIs that don't require authentication:

- **exlink-binance-market.collection.json** - Binance public market data APIs
- **exlink-bybit-market.collection.json** - Bybit public market data APIs
- **exlink-gate-market.collection.json** - Gate public market data APIs
- **exlink-okx-market.collection.json** - OKX public market data APIs

### Trading Collections

Trading collections contain private APIs for order management that require authentication:

- **exlink-binance-trade.collection.json** - Binance trading APIs (orders, trades)
- **exlink-bybit-trade.collection.json** - Bybit trading APIs (orders)
- **exlink-gate-trade.collection.json** - Gate trading APIs (orders)
- **exlink-okx-trade.collection.json** - OKX trading APIs (orders)

### Account Collections

Account collections contain private APIs for account management that require authentication:

- **exlink-binance-account.collection.json** - Binance account APIs (balance, account info)
- **exlink-bybit-account.collection.json** - Bybit account APIs (wallet balance, positions)
- **exlink-gate-account.collection.json** - Gate account APIs (balance)
- **exlink-okx-account.collection.json** - OKX account APIs (balance, positions)

## Environments

Each exchange has two environments:

- **Mainnet** - Production environment (real trading)
- **Testnet** - Test/sandbox environment (simulated trading)

### Environment Variables

Each environment file contains the following variables:

**Binance:**
- `binance_base_url` - API base URL (automatically set based on environment)
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

## Signature Scripts

Authentication scripts are available in the `postman/scripts/` directory. These scripts need to be manually copied into Postman collections' Pre-request Script tabs:

- **sign-binance.js** - Binance HMAC-SHA256 signature
- **sign-bybit.js** - Bybit HMAC-SHA256 signature (v5 API format)
- **sign-gate.js** - Gate HMAC-SHA512 signature
- **sign-okx.js** - OKX HMAC-SHA256 signature (Base64 encoded)

See the "Add Authentication Scripts to Collections" section above for detailed instructions on how to use these scripts.

## Usage Examples

### Example 1: Get Market Data (No Authentication)

1. Select **ExLink - Binance Market Data** collection
2. Select **ExLink - Binance Testnet** environment (or any environment)
3. Choose **Get 24hr Ticker** request
4. Click **Send**

### Example 2: Query Account Balance (Requires Authentication)

1. Ensure you've configured API keys in the environment
2. **Important:** Make sure you've added the authentication script to the collection (see "Add Authentication Scripts to Collections" above)
3. Select **ExLink - Binance Account** collection
4. Select **ExLink - Binance Testnet** environment
5. Choose **Get Account Info** request
6. Click **Send**
7. The Pre-request Script will automatically generate the signature

### Example 3: Create an Order

1. Select the appropriate trading collection
2. Select the correct environment with API keys configured
3. Choose **Create Order** request
4. Modify the request body parameters (symbol, side, quantity, etc.)
5. Click **Send**

## API Endpoints

### Binance

**Market Data:**
- Get Exchange Info
- Get 24hr Ticker
- Get Klines
- Get Recent Trades

**Account:**
- Get Account Info

**Trading:**
- Create Order
- Query Order
- Get Open Orders
- Get My Trades

### Bybit

**Market Data:**
- Get Instruments Info
- Get Tickers
- Get Klines
- Get Recent Trades

**Account:**
- Get Wallet Balance
- Get Positions

**Trading:**
- Create Order
- Query Order

### Gate

**Market Data:**
- Get Currency Pairs
- Get Tickers
- Get Candlesticks

**Account:**
- Get Account Balance

**Trading:**
- Create Order
- Query Order
- Get Open Orders

### OKX

**Market Data:**
- Get Instruments
- Get Ticker
- Get Candles
- Get Recent Trades

**Account:**
- Get Account Balance
- Get Positions

**Trading:**
- Create Order
- Query Order
- Get Pending Orders

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
- **Authentication scripts must be manually added to collections** - Copy the script content from `postman/scripts/` into the collection's Pre-request Script tab
- Market data collections don't require authentication and can be used immediately
- Account and Trading collections require API keys to be configured in the environment and authentication scripts to be added

