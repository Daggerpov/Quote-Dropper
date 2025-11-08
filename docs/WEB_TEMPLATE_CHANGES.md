# Web Template REST Refactoring

This document summarizes the changes made to the web templates to use the new REST API endpoints.

## Changes Made

### 1. `templates/index.html.tmpl`

Updated all category and filter links to use query parameters instead of path-based routing:

**Old URLs:**
```html
/quotes/maxQuoteLength=100
/quotes/classification=motivation
/quotes/classification=wisdom
/quotes/classification=philosophy
/quotes/classification=inspiration
/quotes/classification=discipline
/quotes/classification=love
/quotes/classification=upliftment
```

**New URLs:**
```html
/quotes?maxLength=100
/quotes?classification=motivation
/quotes?classification=wisdom
/quotes?classification=philosophy
/quotes?classification=inspiration
/quotes?classification=discipline
/quotes?classification=love
/quotes?classification=upliftment
```

### 2. `templates/quotes.html.tmpl`

Updated the quote browsing template in three areas:

#### A. Static Filter Links

**Old:**
```html
<a href="/quotes/maxQuoteLength=100" class="nav-link">Short Quotes (≤100)</a>
<a href="/quotes/maxQuoteLength=60" class="nav-link">Very Short (≤60)</a>
```

**New:**
```html
<a href="/quotes?maxLength=100" class="nav-link">Short Quotes (≤100)</a>
<a href="/quotes?maxLength=60" class="nav-link">Very Short (≤60)</a>
```

#### B. Dynamic Category Link Generation (JavaScript)

**Old:**
```javascript
link.href = `/quotes/classification=${category}`;
```

**New:**
```javascript
link.href = `/quotes?classification=${category}`;
```

#### C. URL Parsing and Filter Detection (JavaScript)

**Old - Path-based parsing:**
```javascript
const path = window.location.pathname;

if (path.includes('/maxQuoteLength=60')) {
    currentLengthFilter = '60';
} else if (path.includes('/maxQuoteLength=100')) {
    currentLengthFilter = '100';
}

if (path.includes('/classification=')) {
    const categoryMatch = path.match(/\/classification=([^\/]+)/);
    if (categoryMatch) {
        currentCategoryFilter = categoryMatch[1];
    }
}
```

**New - Query parameter parsing:**
```javascript
const urlParams = new URLSearchParams(window.location.search);

const maxLength = urlParams.get('maxLength');
if (maxLength === '60') {
    currentLengthFilter = '60';
} else if (maxLength === '100') {
    currentLengthFilter = '100';
}

const classification = urlParams.get('classification');
if (classification) {
    currentCategoryFilter = classification;
}
```

#### D. URL Generation for Filter Clicks (JavaScript)

**Old - Path concatenation:**
```javascript
let newPath = '/quotes';

if (currentCategoryFilter !== 'all') {
    newPath += `/classification=${currentCategoryFilter}`;
}

if (currentLengthFilter !== 'all') {
    newPath += `/maxQuoteLength=${currentLengthFilter}`;
}

window.location.href = newPath;
```

**New - Query parameter building:**
```javascript
const params = new URLSearchParams();

if (currentCategoryFilter !== 'all') {
    params.append('classification', currentCategoryFilter);
}

if (currentLengthFilter !== 'all') {
    params.append('maxLength', currentLengthFilter);
}

const queryString = params.toString();
window.location.href = queryString ? `/quotes?${queryString}` : '/quotes';
```

## Benefits

1. **Standards Compliant**: URLs now follow REST conventions with query parameters for filtering
2. **Cleaner URLs**: More intuitive structure (e.g., `/quotes?classification=wisdom&maxLength=100`)
3. **Better Flexibility**: Multiple filters can be combined naturally with `&` in query string
4. **Consistent**: Web UI matches the API structure used by mobile apps

## Example URL Transformations

| Old URL | New URL |
|---------|---------|
| `/quotes` | `/quotes` (unchanged) |
| `/quotes/maxQuoteLength=100` | `/quotes?maxLength=100` |
| `/quotes/classification=wisdom` | `/quotes?classification=wisdom` |
| `/quotes/classification=wisdom/maxQuoteLength=100` | `/quotes?classification=wisdom&maxLength=100` |

## Testing Checklist

Test the following on the live site:

- [ ] Homepage loads correctly
- [ ] All category links work from homepage
- [ ] Short quotes filter works
- [ ] Category filters work in quotes page
- [ ] Length filters work in quotes page
- [ ] Combined filters work (category + length)
- [ ] Filter buttons show active state correctly
- [ ] Client-side search still works
- [ ] Copy quote button works
- [ ] All statistics display correctly

## Deployment

After deploying, test these URLs:

```bash
# Homepage
https://quote-dropper-production.up.railway.app/

# All quotes
https://quote-dropper-production.up.railway.app/quotes

# Short quotes
https://quote-dropper-production.up.railway.app/quotes?maxLength=100

# Category filter
https://quote-dropper-production.up.railway.app/quotes?classification=wisdom

# Combined filters
https://quote-dropper-production.up.railway.app/quotes?classification=wisdom&maxLength=100
```

## Backward Compatibility

**Note**: The old URL format is NO LONGER supported. Users bookmarking old URLs will need to update them. Consider adding redirects if this is a concern:

```go
// Optional redirect middleware for backward compatibility
r.GET("/quotes/maxQuoteLength=:length", func(c *gin.Context) {
    length := c.Param("length")
    c.Redirect(http.StatusMovedPermanently, "/quotes?maxLength="+length)
})

r.GET("/quotes/classification=:classification", func(c *gin.Context) {
    classification := c.Param("classification")
    c.Redirect(http.StatusMovedPermanently, "/quotes?classification="+classification)
})
```

## Files Modified

1. `/Users/daggerpov/Documents/GitHub/Quote-Dropper/src/api/templates/index.html.tmpl`
2. `/Users/daggerpov/Documents/GitHub/Quote-Dropper/src/api/templates/quotes.html.tmpl`

## Next Steps

1. Deploy updated templates to production
2. Test all URLs and filters
3. Monitor for any issues
4. Update any external documentation referencing old URLs
5. Consider adding URL redirects for bookmarked old URLs

