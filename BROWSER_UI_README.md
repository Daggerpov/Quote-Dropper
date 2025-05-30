# Quote Dropper - Browser UI Enhancement

## Overview

The Quote Dropper API now provides a beautiful web interface when accessed from web browsers, while maintaining full JSON API compatibility for the Quote Droplet mobile app.

## How It Works

The system automatically detects whether a request is coming from:
- **Web Browser**: Serves a beautiful HTML interface with styled quote cards
- **Mobile App (Quote Droplet)**: Serves JSON responses as before

## Detection Logic

The system identifies browser requests by checking:
1. **Accept Header**: Must include `text/html`
2. **User Agent**: Must NOT contain:
   - "Quote Droplet" (your app's identifier)
   - "okhttp" (common Android HTTP library)
   - "CFNetwork" (common iOS HTTP library)

## Supported Endpoints with Browser UI

All quote GET endpoints now support browser UI:

### Main Quote Endpoints
- `/quotes` - All approved quotes
- `/quotes/maxQuoteLength=:maxQuoteLength` - Quotes below certain length
- `/quotes/from/:id` - Quotes starting from specific ID
- `/quotes/recent/:limit` - Recent quotes (limit 1-5)
- `/quotes/:id` - Specific quote by ID

### Classification Endpoints
- `/quotes/classification=:classification` - Quotes by category
- `/quotes/classification=:classification/maxQuoteLength=:maxQuoteLength` - Category + length filter
- `/quotes/randomQuote/classification=:classification` - Random quote from category

### Author Endpoints
- `/quotes/author=:author` - All quotes by author
- `/quotes/author=:author/index=:index` - Specific quote by author and index

## Browser UI Features

### Beautiful Design
- Modern gradient background
- Card-based quote layout
- Responsive grid system
- Hover animations and effects
- Mobile-friendly responsive design

### Quote Cards Display
- Quote text with elegant typography
- Author attribution
- Category/classification badges
- Like counts with heart icons
- Quote ID numbers
- Large decorative quotation marks

### Navigation
- Quick access links to popular endpoints
- Filter shortcuts (short quotes, recent quotes)
- Category browsing links
- Breadcrumb-style navigation

### Statistics
- Quote count display
- Filter information (max length, category)
- Search criteria summary

## API Compatibility

**Important**: This enhancement is completely backward compatible. Your Quote Droplet mobile app will continue to receive JSON responses exactly as before. No changes are needed to your mobile app code.

## Example Usage

### For Mobile App (JSON Response)
```bash
curl -H "User-Agent: Quote Droplet/1.0" \
     -H "Accept: application/json" \
     https://your-api.com/quotes/maxQuoteLength=100
```

### For Web Browser (HTML Response)
Simply visit in any web browser:
```
https://your-api.com/quotes/maxQuoteLength=100
```

## Customization

### User Agent Detection
To customize which user agents are treated as mobile apps, modify the `isBrowserRequest()` function in `quotes.go`:

```go
func isBrowserRequest(c *gin.Context) bool {
    accept := c.GetHeader("Accept")
    userAgent := c.GetHeader("User-Agent")
    
    acceptsHTML := strings.Contains(accept, "text/html")
    
    // Customize this logic for your app
    isNotMobileApp := !strings.Contains(userAgent, "Your App Name") && 
                      !strings.Contains(userAgent, "okhttp") && 
                      !strings.Contains(userAgent, "CFNetwork")
    
    return acceptsHTML && isNotMobileApp
}
```

### Styling
The HTML template is located at `src/api/templates/quotes.html.tmpl`. You can customize:
- Colors and gradients
- Typography and fonts
- Layout and spacing
- Animation effects
- Responsive breakpoints

## Error Handling

The browser UI includes user-friendly error pages for:
- Invalid parameters
- Database errors
- Not found errors
- Server errors

All errors maintain the same visual design as the quote display pages.

## Performance

- Templates are loaded once at startup
- Minimal overhead for request detection
- Same database queries as JSON API
- Efficient HTML rendering with Go templates 