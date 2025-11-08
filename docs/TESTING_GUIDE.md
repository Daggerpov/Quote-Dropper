# Testing Guide for REST API Changes

This guide provides step-by-step instructions for testing the new REST API endpoints in both the Go backend and iOS app.

## Prerequisites

- Go 1.16+ installed
- Xcode 14+ installed
- PostgreSQL database running
- Environment variables set (DATABASE_URL, ADMIN_USERNAME, ADMIN_PASSWORD)

## Backend Testing

### 1. Build the API

```bash
cd /Users/daggerpov/Documents/GitHub/Quote-Dropper/src/api
go build -o quote-dropper .
```

### 2. Run the API Locally

```bash
# Set environment variables
export DATABASE_URL="your_database_url"
export PORT=8080
export ADMIN_USERNAME="admin"
export ADMIN_PASSWORD="your_password"
export GIN_MODE="debug"

# Run the server
./quote-dropper
```

### 3. Test REST Endpoints

#### Test Quote Retrieval

```bash
# Get all quotes
curl -X GET http://localhost:8080/quotes

# Get random quote
curl -X GET "http://localhost:8080/quotes/random"

# Get random wisdom quote
curl -X GET "http://localhost:8080/quotes/random?classification=wisdom"

# Get short random quote
curl -X GET "http://localhost:8080/quotes/random?maxLength=65"

# Get quotes by classification
curl -X GET "http://localhost:8080/quotes?classification=wisdom"

# Get short wisdom quotes
curl -X GET "http://localhost:8080/quotes?classification=wisdom&maxLength=100"

# Get quotes by author
curl -X GET "http://localhost:8080/quotes?author=Socrates"

# Search quotes
curl -X GET "http://localhost:8080/quotes?search=happiness"

# Search in specific category
curl -X GET "http://localhost:8080/quotes?search=happiness&category=wisdom"

# Get recent quotes
curl -X GET "http://localhost:8080/quotes/recent?limit=5"

# Get top quotes
curl -X GET "http://localhost:8080/quotes/top?category=all"

# Get quote count
curl -X GET "http://localhost:8080/quotes/count?category=all"

# Get specific quote by ID
curl -X GET "http://localhost:8080/quotes/1"
```

#### Test Quote Creation

```bash
curl -X POST http://localhost:8080/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Test quote for REST API",
    "author": "Test Author",
    "classification": "wisdom",
    "submitter_name": "Tester"
  }'
```

#### Test Like/Unlike

```bash
# Like a quote
curl -X POST http://localhost:8080/quotes/1/like

# Unlike a quote
curl -X DELETE http://localhost:8080/quotes/1/like
```

#### Test Feedback

```bash
curl -X POST http://localhost:8080/feedback \
  -H "Content-Type: application/json" \
  -d '{
    "type": "general",
    "name": "Test User",
    "content": "Test feedback message"
  }'
```

#### Test Categories

```bash
curl -X GET http://localhost:8080/categories
```

### 4. Test Admin Endpoints

```bash
# View unapproved quotes (requires auth)
curl -X GET http://localhost:8080/admin \
  -u admin:your_password

# Search quotes (admin)
curl -X GET "http://localhost:8080/admin/quotes/search?q=happiness" \
  -u admin:your_password

# Approve a quote
curl -X POST http://localhost:8080/admin/quotes/1/approve \
  -u admin:your_password

# Edit a quote
curl -X PUT http://localhost:8080/admin/quotes/1 \
  -u admin:your_password \
  -H "Content-Type: application/json" \
  -d '{
    "edit_text": "Updated quote text",
    "edit_author": "Updated Author",
    "edit_classification": "wisdom"
  }'

# Delete a quote
curl -X DELETE http://localhost:8080/admin/quotes/1 \
  -u admin:your_password

# View feedback
curl -X GET http://localhost:8080/admin/feedback \
  -u admin:your_password

# Delete feedback
curl -X DELETE http://localhost:8080/admin/feedback/1 \
  -u admin:your_password

# View migrations
curl -X GET http://localhost:8080/admin/migrations \
  -u admin:your_password
```

## iOS App Testing

### 1. Open the Project

```bash
open "/Users/daggerpov/Documents/GitHub/Quote-Droplet-iOS/Quote Droplet.xcworkspace"
```

### 2. Update Base URL (if testing locally)

Edit `APIService.swift` and change the base URL:

```swift
private let baseUrl = "http://localhost:8080"  // For local testing
// private let baseUrl = "https://quote-dropper-production.up.railway.app"  // Production
```

### 3. Build and Run

1. Select a simulator or device
2. Press Cmd+R to build and run
3. Check the console for API logs

### 4. Test Features

#### Test Random Quotes
1. Open the app
2. Tap "Get Random Quote"
3. Verify quotes are fetched
4. Check console logs for successful API calls

#### Test Categories
1. Navigate to category selection
2. Select different categories
3. Verify filtered quotes are displayed
4. Check for proper API calls in console

#### Test Search
1. Navigate to search screen
2. Enter a search term
3. Verify search results appear
4. Test with different categories

#### Test Author View
1. Tap on a quote's author
2. Verify all quotes by that author are displayed
3. Check navigation works correctly

#### Test Like/Unlike
1. Tap the like button on a quote
2. Verify the like count increases
3. Tap again to unlike
4. Verify the like count decreases
5. Check console for proper DELETE request

#### Test Submit Quote
1. Navigate to submit quote screen
2. Fill in quote details
3. Submit the quote
4. Verify success message

#### Test Feedback
1. Navigate to feedback screen
2. Fill in feedback form
3. Submit feedback
4. Verify success message

#### Test Recent Quotes
1. Navigate to recent quotes
2. Verify the most recent quotes are displayed
3. Check the limit parameter is respected

## Verification Checklist

### Backend
- [ ] All REST endpoints compile without errors
- [ ] GET requests return proper JSON responses
- [ ] POST requests create new resources
- [ ] DELETE requests remove resources
- [ ] PUT requests update resources
- [ ] Query parameters are correctly parsed
- [ ] Error responses are properly formatted
- [ ] Authentication works for admin endpoints
- [ ] Rate limiting is functioning

### iOS App
- [ ] App compiles without errors
- [ ] No crashes on launch
- [ ] Random quotes load correctly
- [ ] Search functionality works
- [ ] Author filtering works
- [ ] Like/unlike works with new DELETE method
- [ ] Feedback submission succeeds
- [ ] Recent quotes display correctly
- [ ] Top quotes display correctly
- [ ] Quote counts are accurate
- [ ] All API calls use new endpoints

## Common Issues and Solutions

### Issue: "Connection refused"
**Solution:** Ensure the backend server is running on the correct port

### Issue: "Invalid URL"
**Solution:** Check that query parameters are properly URL encoded

### Issue: "404 Not Found"
**Solution:** Verify the endpoint path matches the new REST conventions

### Issue: "401 Unauthorized" (Admin endpoints)
**Solution:** Check that admin credentials are correctly set in environment variables

### Issue: iOS app shows old data
**Solution:** Clean build folder (Cmd+Shift+K) and rebuild

### Issue: Likes not updating correctly
**Solution:** Verify the DELETE method is being used for unlike (not POST)

## Performance Testing

### Load Testing

Use `ab` (ApacheBench) to test endpoint performance:

```bash
# Test quote retrieval
ab -n 1000 -c 10 http://localhost:8080/quotes

# Test random quotes
ab -n 1000 -c 10 http://localhost:8080/quotes/random

# Test search
ab -n 1000 -c 10 "http://localhost:8080/quotes?search=happiness"
```

### Monitor Logs

Watch the server logs for errors:

```bash
tail -f /var/log/quote-dropper.log
```

## Rollback Plan

If issues are encountered in production:

1. **Keep the old binary:**
   ```bash
   cp quote-dropper quote-dropper.backup
   ```

2. **Revert to previous version:**
   ```bash
   git checkout <previous-commit>
   go build -o quote-dropper .
   ```

3. **Restore iOS app:**
   ```bash
   git checkout <previous-commit> QuoteDroplet/Services/APIService/APIService.swift
   ```

## Next Steps

1. Deploy to staging environment
2. Run full test suite
3. Monitor logs for errors
4. Get user feedback
5. Deploy to production
6. Update API documentation
7. Consider implementing API versioning

## Support

For issues or questions:
- Check the API_REST_CHANGES.md file
- Review server logs
- Check iOS console output
- Consult REST API best practices

## Useful Commands

```bash
# Check server status
curl -I http://localhost:8080/quotes

# Pretty print JSON responses
curl http://localhost:8080/quotes | jq

# Save response to file
curl http://localhost:8080/quotes > quotes.json

# Measure response time
time curl http://localhost:8080/quotes/random

# Check specific HTTP status
curl -w "%{http_code}\n" -o /dev/null -s http://localhost:8080/quotes/1
```

