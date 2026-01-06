##

# 1.1 File Storage. Welcome
Welcome
Welcome to large file storage! Building a (good) web application almost always involves handling "large" files of some kind - whether its static images and videos for a marketing site, or user generated content like profile pictures and video uploads, it always seems to come up.

In this course we'll cover strategies for handling files that are kilobytes, megabytes, or even gigabytes in size, as opposed to the small structured data that you might store in a traditional database (integers, booleans, and simple strings).

Learning Goals
Understand what "large" files are and how they differ from "small" structured data
Build an app that uses AWS S3 and Go to store and serve assets
Learn how to manage files on a "normal" (non-s3) filesystem based application
Learn how to store and serve assets at scale using serverless solutions, like AWS S3
Learn how to stream video and to keep data usage low and improve performance
AWS Account Required
This course will require an AWS account. We will not go outside of the free tier, so if you do everything properly you shouldn't be charged. That said, you will need to have a credit card on file, and if you do something wrong you could be charged, so just be careful and understand the risk.

We recommend deleting all the resources that you create when you're done with the course to avoid any charges. We'll remind you at the end.

Tubely
In this course we'll be building "Tubely", a SaaS product that helps YouTubers manage their video assets. It allows users to upload, store, serve, add metadata to, and version their video files. It will also allow them to manage thumbnails, titles, and other video metadata.

Assignment
You'll need both the Go toolchain (version 1.25+) and the Boot.dev CLI installed. If you don't already have them, here are the installation instructions.
Fork the starter repo for this course into your own GitHub namespace, then clone your fork onto your local machine.
Copy the .env.example file to .env. In the future we'll edit some of the .env values to match your configuration, but for now you can leave them as is.
cp .env.example .env

Run the Tubely server:
go run .

A URL will be logged to the console, open the URL in a browser to see the Tubely app. The webpage should load, but don't try to interact with it yet.

Run and submit the CLI tests with the server running.

Troubleshooting
If you get an error that says "go-sqlite3 requires cgo to work", you need to:

Install gcc:
on macOS:

brew install gcc

or Linux:

sudo apt install gcc

Ensure the environment variable CGO_ENABLED is set to 1:
go env CGO_ENABLED

# If the command above prints 0, run this

go env -w CGO_ENABLED=1

# 1.2 Large Files

So, what are "large files" anyway?

Click to hide video

You're probably already familiar with small structured data; the stuff that's usually stored in a relational database like Postgres or MySQL. I'm talking about simple, primitive data types like:

user_id (integer)
is_active (boolean)
email (string)
Large files, or "large assets", on the other hand, are giant blobs of data encoded in a specific file format and measured in kilo, mega, or gigabytes. As a simple rule:

If it makes sense to go into an excel spreadsheet, it probably belongs in a traditional database
If it would normally be stored on your hard drive as its own file, it probably is a "large file"
Large files are interesting because:

They're large in size (duh) and are thus more performance-sensitive
They're usually accessed frequently, and this combined with their size can quickly lead to performance bottlenecks
Assignment
In the root of your repo there is a script called samplesdownload.sh. Run it from the root of the repo to download some sample images and videos into the samples directory:
./samplesdownload.sh

Note that this will download about 60MB of sample files, so depending on your internet connection it might take a second.

Take a look at the boots-image-horizontal.png file in the samples directory: it's a PNG image file. You can open it in an image viewer to see what it looks like.
Use xxd to view a hexdump of the samples/boots-image-horizontal.png file:
xxd <file>

xxd converts the binary content of the file into a human-readable hexadecimal and ASCII formats. You should see a bunch of gobbledegook - that's what a PNG file looks like in its raw form.

Inspect the first 8 bytes of the file more closely. Use xxd with -l (length) to limit the output:
xxd -l 8 <file>

You'll see that the first 8 bytes are 89 50 4e 47 0d 0a 1a 0a, which is the PNG file signature. It tells the reader that this is a PNG file, in fact, the characters PNG are present in bytes 2-4.

# 1.3 Database

Tubely's architecture is simple. We're using:

Golang to write the application
SQLite as a database. SQLite is a traditional relational database that works out of a single flat file, meaning it doesn't need a separate server process to run.
Later, we'll also use the filesystem and S3 to store large files
However, small structured data, like user records, will always live in the SQLite database.

Assignment
With the server running, create a Tubely account by entering the following email and password and clicking "sign up":
Email: <admin@tubely.com>
Password: password
We'll use these credentials for the entire course!

Install SQLite 3. The application is already set up to use SQLite, I just want you to be able to use the CLI to manually inspect the database.

- linux

sudo apt update
sudo apt install sqlite3

- mac

brew update
brew install sqlite3

Run sqlite3 tubely.db to open the database file (that should have been created automatically by the server when you started it). Run a select * from users; to see the users table. You should see yourself in there!
Type .exit to exit the SQLite CLI.

# 1.4 Videos

The two main entities in Tubely are videos and users. A user can have many videos, and a video belongs to a single user.

"Videos" have 3 things to worry about:

Metadata: The title, description, and other information about the video
Thumbnail: An image that represents the video
Video: The actual video file
Tubely allows users to create a "new draft" - which creates a new video record in the database containing metadata only. Thumbnails and video files are uploaded separately after the draft is created.

Assignment
Create a new video with the following:
Title: "Boots, an Emote Story"
Description: "A short film about the many faces of Boots"
You should now see the video in the UI with options to upload a thumbnail and video file. Don't upload the files yet.

# 1.5 Multipart Uploads

Let's work with some files already!

So you might already be familiar with simple JSON/HTML form POST requests. That works great for small structured data (small strings, integers, etc.), but what about large files?

We don't typically send massive files as single JSON payloads or forms. Instead, we use a different encoding format called multipart/form-data. In a nutshell, it's a way to send multiple pieces of data in a single request and is commonly used for file uploads. It's the "default" way to send files to a server from an HTML form.

Luckily, the Go standard library's net/http package has built-in support for parsing multipart/form-data requests. The http.Request struct has a method called ParseMultipartForm.

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r*http.Request) {
    // validate the request

 const maxMemory = 10 << 20
 r.ParseMultipartForm(maxMemory)

 // "thumbnail" should match the HTML form input name
 file, header, err := r.FormFile("thumbnail")
 if err != nil {
  respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
  return
 }
 defer file.Close()

 // `file` is an `io.Reader` that we can read from to get the image data

Assignment
The handler for uploading thumbnails is currently a no-op. Let's get it working. We're going to keep it simple and store all image data in-memory.

Notice that in main.go there is a global map of video IDs to thumbnail structs called videoThumbnails. This is where we're going to store the thumbnail data.
Notice the handlerThumbnailGet function. It serves the thumbnail file back to the UI, but it assumes that images exist in the videoThumbnails map (which they don't yet!)
Complete the handlerUploadThumbnail function. It handles a multipart form upload of a thumbnail image and stores it in the videoThumbnails map:

Authentication has already been taken care of for you, and the video's ID has been parsed from the URL path.
Parse the form data
Set a const maxMemory to 10MB. I just bit-shifted the number 10 to the left 20 times to get an int that stores the proper number of bytes.
Use (http.Request).ParseMultipartForm with the maxMemory const as an argument
Bit shifting is a way to multiply by powers of 2. 10 << 20 is the same as 10 *1024* 1024, which is 10MB.
Get the image data from the form
Use r.FormFile to get the file data and file headers. The key the web browser is using is called "thumbnail"
Get the media type from the form file's Content-Type header
Read all the image data into a byte slice using io.ReadAll
Get the video's metadata from the SQLite database. The apiConfig's db has a GetVideo method you can use
If the authenticated user is not the video owner, return a http.StatusUnauthorized response
Save the thumbnail to the global map
Create a new thumbnail struct with the image data and media type
Add the thumbnail to the global map, using the video's ID as the key
Update the video metadata so that it has a new thumbnail URL, then update the record in the database by using the cfg.db.UpdateVideo function. The thumbnail URL should have this format:
<http://localhost>:<port>/api/thumbnails/{videoID}

This will all work because the /api/thumbnails/{videoID} endpoint serves thumbnails from that global map.

Respond with updated JSON of the video's metadata. Use the provided respondWithJSON function and pass it the updated database.Video struct to marshal.
Test your handler manually by using the Tubely UI to upload the boots-image-horizontal.png image. You should see the thumbnail update in the UI!

# 1.6 Encoding

As you know, we use a SQLite database to power the majority of the web app. SQLite is a traditional relational database that works out of a single flat file, meaning it doesn't need a separate server process to run.

Let's talk about the elephant in the room: Our current solution for video thumbnails (storing the media in-memory) is a terrible solution. If the server is restarted, all the thumbnails are lost!

But we can't store an image in a SQLite column?... Right?

hold my beer
To do so, we can actually encode the image as a base64 string and shove the whole thing into a text column in SQLite. Base64 is just a way to encode binary (raw) data as text. It's not the most efficient way to do it, but it will work for now.

Assignment
Update the code to store the image data in the thumbnail_url column in the database.

Use base64.StdEncoding.EncodeToString from the encoding/base64 package to convert the image data to a base64 string.
Create a data URL with the media type and base64 encoded image data. The format is:
data:<media-type>;base64,<data>

Store the URL in the thumbnail_url column in the database.
Because the thumbnail_url has all the data we need, delete the global thumbnail map and the GET route for thumbnails.
Restart the server and re-upload the boots-image-horizontal.png thumbnail image to ensure it's working.

# 1.7 Using the Filesystem

Great, now we're using base64 strings in our SQLite database to store images... let's talk about why that actually kinda sucks.

CPU performance: Base64 encoding is an expensive CPU-intensive operation. If we have a lot of uploads (I mean, we're planning on being a successful company, right?), we'll have some scaling issues.
Storage costs: Base64 encoding bloats the size of the image data. We're using more disk space than we need to, which again, is expensive and slow.
Database performance: Databases (especially relational databases like SQLite, Postgres and MySQL) are optimized for small, structured data, not giant blobs of binary. It will impact query performance in a non-trivial way.
Caching: Base64 encoded images aren't as cache friendly as raw files, meaning slower load times and higher bandwidth costs.
It's usually a bad idea to store large binary blobs in a database, there are exceptions, but they are rare. So what's the solution? Store the files on the file system. File systems are optimized for storing and serving files, and they do it well.

Assignment
Let's update our handler to store the files on the file system. We'll save uploaded files to the /assets directory on disk.

Instead of encoding to base64, update the handler to save the bytes to a file at the path /assets/<videoID>.<file_extension>.
Use the Content-Type header to determine the file extension.
Use the videoID to create a unique file path. filepath.Join and cfg.assetsRoot will be helpful here.
Use os.Create to create the new file
Copy the contents from the multipart.File to the new file on disk using io.Copy
Update the thumbnail_url. Notice that in main.go we have a file server that serves files from the /assets directory. The URL for the thumbnail should now be:
<http://localhost>:<port>/assets/<videoID>.<file_extension>

Restart the server and re-upload the boots-image-horizontal.png thumbnail image to ensure it's working. You should see it in the UI as well as a copy in the /assets directory.

# 1.8 Mime Types

There are an infinite number of things we could consider "large files". But within the context of web development, the most common types of large files are probably:

Images: PNGs, JPEGs, GIFs, SVGs, etc.
Videos: MP4s, MOVs, AVIs, etc.
Audio: MP3s, WAVs, etc.
Static web templates: HTML, CSS, JS, etc.
Administrative files: PDFs, Word docs, etc.
A mime type is just a web-friendly way to describe format of a file. It's kind of like a file extension, but more standardized and built for the web.

Mime types have a type and a subtype, separated by a /. For example:

image/png
video/mp4
audio/mp3
text/html
When a browser uploads a file via a multipart form, it sends the file's mime type in the Content-Type header.

Assignment
Up until now we've allowed any file type to be uploaded as a thumbnail... let's fix that.

Use the mime.ParseMediaType function to get the media type from the Content-Type header
If the media type isn't either image/jpeg or image/png, respond with an error (respondWithError helper)
Try to upload the PDF file "is-bootdev-for-you.pdf" as a thumbnail - you should get an error

# 1.9 Live Edits

With Tubely, we have it easy. Users won't be able to make small tweaks to existing images and videos - like changing the background color or adding a text overlay.

We'll force them to simply upload new versions of the file (even YouTube has this restriction, even with their resources).

If a user were able to live edit a file (think Google Docs or Canva) we'd have to approach our storage problem differently. We wouldn't just be managing new versions of "static" files, we would need to handle every tiny edit (keystroke) and sync updated changes to our server. That's much more complicated and outside the scope of this course.

Luckily, those requirements are also less common in the real world. The Tubely use case (storing and serving entire assets) is much more common and much easier to implement.

# 2.1 Caching

Tubely is a web application. It's accessed through a browser and browsers love to cache stuff. They love cache more than Scrooge McDuck.

I'll see myself out.
"Cache" is just a fancy word for "temporary storage". When a user visits a web application for the first time, their browser downloads all the files required to display the page: HTML, CSS, JS, images, videos, etc. It then "caches" (stores) them on the user's machine so that next time they come back, it doesn't need to re-download everything. It can use the locally stored copies.

Assignment
Run the server and open the app. Your video should already have the boots-image-horizontal.png uploaded, and you should be able to see it in the web page
On the same video, upload the boots-image-vertical.png image instead.
Refresh the browser

# 2.2 Caching

Tubely is a web application. It's accessed through a browser and browsers love to cache stuff. They love cache more than Scrooge McDuck.

I'll see myself out.
"Cache" is just a fancy word for "temporary storage". When a user visits a web application for the first time, their browser downloads all the files required to display the page: HTML, CSS, JS, images, videos, etc. It then "caches" (stores) them on the user's machine so that next time they come back, it doesn't need to re-download everything. It can use the locally stored copies.

Assignment
Run the server and open the app. Your video should already have the boots-image-horizontal.png uploaded, and you should be able to see it in the web page
On the same video, upload the boots-image-vertical.png image instead.
Refresh the browser


Cache Busting
Click to hide video

Browsers cache stuff for good reason: it makes the user experience snappier and, if the user is paying for data, cheaper.

That said, sometimes (like in the last lesson) we don't want the browser to cache a file - we want to be sure we have the latest version. One trick to ensure that we get the latest is by "busting the cache". A simple tactic is to change the URL of the file a bit. Say we have this image URL:

<http://localhost:8080/image.jpg>

To cache bust, we want to alter the URL so that:

The browser thinks it's a different file
The server thinks it's the same file
Servers typically ignore query strings for file-like assets, so one of the most common ways to cache bust from the client side is to just add one. For example, a version parameter like this:

<http://localhost:8080/image.jpg?version=1>

If we want to bust it again, we just increment the version:

<http://localhost:8080/image.jpg?version=2>

Our use of the version key and the 1 and 2 values are arbitrary. The important thing is that the URL is different.

Assignment
We won't do much with the front-end of Tubely in this course, but this one lesson is an exception.

Open app/app.js and take a look at the viewVideo function. Find this line:
thumbnailImg.src = video.thumbnail_url;

This is where we tell the browser which URL to load the thumbnail image from. Because the URL doesn't change when we upload a new thumbnail, let's add client-side cache busting.

Update the above line of code to instead append a query string with this format:
ORIGINAL_URL?v=TIME

Where ORIGINAL_URL is the original URL and TIME is the current time in milliseconds. You can get the current time in milliseconds with Date.now(). You can also use string interpolation in JS like this:

message = `She is ${age} years old`;

Restart the application, and try switching back and forth between the two thumbnails. You should see the new thumbnail immediately after uploading!
Run and submit the CLI tests from the root of the repo.

You may have to hard refresh and clear your browser's cache to see the changes in app.js
The tests for this step are a bit odd, they're checking your app.js to just make sure that you're not doing the old thing. Don't cheat please and thanks.


Cache Headers
Query strings are a great way to brute force cache controls as the client - but the best way (assuming you have control of the server, and c'mon, we're backend devs), is to use the Cache-Control header. Some common values are:

no-store: Don't cache this at all
max-age=3600: Cache this for 1 hour (3600 seconds)
stale-while-revalidate: Serve stale content while revalidating the cache
no-cache: Does not mean "don't cache this". It means "cache this, but revalidate it before serving it again"
The fact that no-cache means "lol jk you can actually cache this just check with me first" makes me feel some sort of way
You can view all the other options here if you're interested.

When the server sends Cache-Control headers, it's up to the browser to respect them, but most modern browsers do.

Assignment
Let's do things the right way and control caching with headers from the server.

Edit app.js and revert it back to the way it was, just setting thumbnailImg.src = video.thumbnail_url; in the viewVideo function.
In cache.go rename the cacheMiddleware function to noCacheMiddleware (it will be more descriptive now)
Update the noCacheMiddleware function. It should set the Cache-Control header to no-store before handling the request.
Restart the application, and try switching back and forth between the two thumbnails. You should see the new thumbnail immediately after uploading, even without client side cache busting!


# 2.3 Cache Busting

Click to hide video

Browsers cache stuff for good reason: it makes the user experience snappier and, if the user is paying for data, cheaper.

That said, sometimes (like in the last lesson) we don't want the browser to cache a file - we want to be sure we have the latest version. One trick to ensure that we get the latest is by "busting the cache". A simple tactic is to change the URL of the file a bit. Say we have this image URL:

<http://localhost:8080/image.jpg>

To cache bust, we want to alter the URL so that:

The browser thinks it's a different file
The server thinks it's the same file
Servers typically ignore query strings for file-like assets, so one of the most common ways to cache bust from the client side is to just add one. For example, a version parameter like this:

<http://localhost:8080/image.jpg?version=1>

If we want to bust it again, we just increment the version:

<http://localhost:8080/image.jpg?version=2>

Our use of the version key and the 1 and 2 values are arbitrary. The important thing is that the URL is different.

Assignment
We won't do much with the front-end of Tubely in this course, but this one lesson is an exception.

Open app/app.js and take a look at the viewVideo function. Find this line:
thumbnailImg.src = video.thumbnail_url;

This is where we tell the browser which URL to load the thumbnail image from. Because the URL doesn't change when we upload a new thumbnail, let's add client-side cache busting.

Update the above line of code to instead append a query string with this format:
ORIGINAL_URL?v=TIME

Where ORIGINAL_URL is the original URL and TIME is the current time in milliseconds. You can get the current time in milliseconds with Date.now(). You can also use string interpolation in JS like this:

message = `She is ${age} years old`;

Restart the application, and try switching back and forth between the two thumbnails. You should see the new thumbnail immediately after uploading!
Run and submit the CLI tests from the root of the repo.

You may have to hard refresh and clear your browser's cache to see the changes in app.js
The tests for this step are a bit odd, they're checking your app.js to just make sure that you're not doing the old thing. Don't cheat please and thanks.


# 2.4 Cache Headers

Query strings are a great way to brute force cache controls as the client - but the best way (assuming you have control of the server, and c'mon, we're backend devs), is to use the Cache-Control header. Some common values are:

no-store: Don't cache this at all
max-age=3600: Cache this for 1 hour (3600 seconds)
stale-while-revalidate: Serve stale content while revalidating the cache
no-cache: Does not mean "don't cache this". It means "cache this, but revalidate it before serving it again"
The fact that no-cache means "lol jk you can actually cache this just check with me first" makes me feel some sort of way
You can view all the other options here if you're interested.

When the server sends Cache-Control headers, it's up to the browser to respect them, but most modern browsers do.

Assignment
Let's do things the right way and control caching with headers from the server.

Edit app.js and revert it back to the way it was, just setting thumbnailImg.src = video.thumbnail_url; in the viewVideo function.
In cache.go rename the cacheMiddleware function to noCacheMiddleware (it will be more descriptive now)
Update the noCacheMiddleware function. It should set the Cache-Control header to no-store before handling the request.
Restart the application, and try switching back and forth between the two thumbnails. You should see the new thumbnail immediately after uploading, even without client side cache busting!

# 2.5 New Files

"Stale" files are a common problem in web development. And when your app is small, the performance benefits of aggressively caching files might not be worth the complexity and potential bugs that can crop up from not handling cache behavior correctly. After all, the famous quote goes:

There are only two hard things in Computer Science: cache invalidation, naming things, and off-by-one errors.
That said, there is one more strategy I want to cover. It's my personal favorite for apps like Tubely.

In Tubely, we just don't care about old versions of thumbnails. Like ever. So let's just give each new thumbnail version a completely new URL (and path on the filesystem). That way, we can avoid all potential caching issues completely.

It's not that caching is bad generally (it's incredibly useful for many performance-related issues), but we know we don't need it for this part of this app.

Assignment
One final cache update! Each time a new thumbnail is uploaded, we'll give it a new path on disk (and by extension, a new URL). This way, we can avoid all caching issues completely.

Update the handlerUploadThumbnail function.
Instead of using the videoID to create the file path, use crypto/rand.Read to fill a 32-byte slice with random bytes. Use base64.RawURLEncoding to then convert it into a random base64 string. Use this string as the file name, and set the extension based on the media type (same as before). For example:

QmFzZTY0U3RyaW5nRXhhbXBsZQ.png

Test the new functionality by swapping back and forth between two different thumbnail images. Right click on the images in the browser, and use "inspect element" to make sure that the URL changes each time you upload a new thumbnail.
