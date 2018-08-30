# images

## Whats going on here?

So images are actually a difficult problem.
We don't want to use the images provided to us because they are:
1. Too big,
2. From an endpoint that takes too long to load

### The solution
We can throw images into our bucket, which we can put behind a CDN (Cloudflare) and serve those instead. Before putting into the bucket we can resize and minify them too.

This big script, is going through the database looking for unminified images, onces it finds some, it goes through, downloads the image, processes it, then upload it to Spaces, once that's done it writes back to the database the URL.
