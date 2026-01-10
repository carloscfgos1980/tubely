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

# 3.1 Single Machine

I promised you AWS S3, and you're gonna get it. But first, let's understand why S3 being "serverless" is kind of a big deal.

In a "simple" web application architecture, your server is likely a single machine running in the cloud. That single machine probably runs:

An HTTP server that handles the incoming requests
A database running in the background that the HTTP server talks to
A file system the server uses to directly read and write larger files

All on one machine.

What's Wrong With This?
Well, not much. Honestly this is a perfectly valid way to build a web application, even in production. That said, there are some trade-offs:

Scaling: If your app gets popular, you'll need to "scale" your single machine (add more resources like CPU/RAM/Disk space). A single computer can only become so powerful.
Availability: If your server goes down, your app goes down. To be fair, you can mitigate this with load balancers and multiple servers.
Durability: If your server crashes, or an intern rm -rf's something, you're in trouble. You might have backups, but let's be honest, you probably don't.
Cost: Running a server 24/7 means paying 24/7. It can be nice to only pay for what you use.
Maintenance: You have to manage everything yourself. You'll be responsible for "ops" tasks like backups, monitoring, logging, version upgrades, etc.


# 3.2 AWS

AWS (Amazon Web Services) is one of the (at least in my mind) "Big Three" cloud providers. The other two are Google Cloud and Microsoft Azure.

AWS is the oldest, largest, and most popular of the three, generally speaking.

Security best practice: Only use your AWS account's root user to perform a few account and service management tasks. Do NOT use the root user for everyday tasks. After you create your account, secure the root user with a strong password and enable multi-factor authentication (2FA) on it.
For all day-to-day administrative work, create a new IAM user with admin permissions. This will help keep your account secure.
Assignment
Let's get you up and running with AWS.

Create an AWS account (if you don't already have one)
Go to AWS
Create an Account
Select "personal" account
Fill out the form
Add billing info: You won't be charged if you stay within the free tier, which is all you'll need for this course - a $1 hold will be placed on your card to verify it's real, but it will be refunded in a few days
Verify phone number
Use the free support plan
Create an IAM user
Create an iam user in the AWS console
Name it after you
No need for console access
Make a new managers user group with full admin access and attach the user to it
Create the user
Install the AWS CLI version 2
Authorize the IAM user through the CLI
Select the user and click "create access key"
Select "CLI" and ignore the recommendations
Leave tag value blank
Run aws configure
Enter the access key and secret key
Leave region/format blank
No need to download the keys.
Verify CLI sign in with aws sts get-caller-identity
cat ~/.aws/credentials to see the keys


# 3.3 Serverless

Click to hide video

"Serverless" is an architecture (and let's be honest, a buzzword) that refers to a system where you don't have to manage the servers on your own.

Serverless is largely misunderstood due to the dubious naming. It does not mean there are no servers, it just means they're someone else's problem.
You'll often see "Serverless" used to describe services like AWS Lambda, Google Cloud Functions, and Azure Functions. And that's true, but it refers to "serverless" in its most "pure" form: serverless compute.

AWS S3 was actually one of the first "serverless" services, and is arguably still the most popular. It's not serverless compute, it's serverless storage. You don't have to manage/scale/secure the servers that store your files, AWS does that for you.

Instead of going to a local file system, your server makes network requests to the S3 API to read and write files.

Assignment
Make a S3 bucket through the AWS console called tubely-<random_number>. Replace <random_number> with a "random" (just choose one) number to the end of the bucket name to ensure it's unique. For example, tubely-56841, but choose your own number.
The bucket name has to be tubely-<random_number> to pass the tests.
Uncheck "Block all public access" when creating the bucket.
Leave bucket versioning off.
Leave default encryption on with managed keys.
Leave object lock disabled.
Use the AWS CLI to ensure the bucket is there.

aws s3 ls

Now that we have a bucket, we still need to configure its permissions to control who can access the files and how, typically through a bucket policy or object-level ACLs.

Go to your bucket and configure the bucket policy (under the "Permissions" tab). Copy-paste the following, replacing BUCKET_NAME with your bucket.
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::BUCKET_NAME/*"
    }
  ]
}

This will give read-only access to anyone with the precise URL of an object. Notably, this doesn't allow listing the objects in the bucket, only reading them if you know the exact URL.

# 3.4 Upload Object

Now that you have a public bucket, let's add a file.

Assignment
Upload the boots-image-horizontal.png image file to your newly created bucket
Use default settings
Open the file in a browser using the file's URL to see it
Use the aws CLI to check for the file and redirect the output to /tmp/bucket_contents.txt
aws s3 ls BUCKET_NAME > /tmp/bucket_contents.txt

Replace BUCKET_NAME with the name of your bucket

cat the file to make sure you can see your file
cat /tmp/bucket_contents.txt


# 3.5 Architecture

S3 is really simple.

Honestly idk why you even bought this course...
File A goes in bucket B at key C. That's it. You only need 2 things to access an object in S3:

The bucket name
The object key

Buckets have globally unique names because they are part of the URL used to access them. If I make a bucket called "bd-vids", you can't make a bucket called "bd-vids", even if you're in a separate AWS account. This makes it really easy to think about where your data lives.

Assignment
Try to make a bucket called bootdev. It won't work. Ha! I took that name first. A lot of organizations use a company specific prefix to ensure their bucket names are unique. For example, bootdev-user-images.


# 3.6 SDKs and S3

An SDK or "Software Development Kit" is just a collection of tools (often involving an importable library) that helps you interact with a specific service or technology.

AWS has official SDKs for most popular programming languages. They're usually the best way to interact with AWS services.

Don't roll your own crypto, and don't roll your own AWS SDK.
When you as a human interact with AWS resources, you'll typically use the web console (GUI) or the CLI. When your code interacts with AWS resources, you'll use the SDK within your code.

Assignment
You've escaped writing code for a while - this one will have you write a lot of code and might take a bit of time, that's okay!

Install the AWS S3 Go SDK
go get github.com/aws/aws-sdk-go-v2/service/s3 github.com/aws/aws-sdk-go-v2/config

Configure an S3 client in main.go
Add an s3Client field to apiConfig of type *s3.Client
Use config.LoadDefaultConfig to auto load the default AWS SDK config (the keys you set with aws configure)
As arguments, give it an empty Context and pass config.WithRegion(s3Region) to use the region that's set in your .env file.
Create a client with your config using s3.NewFromConfig
Assign the client to the s3Client field
Update the S3_BUCKET and S3_REGION variables in your .env file. They'll be saved into your apiConfig.
Complete the (currently empty) handlerUploadVideo handler to store video files in S3. Images will stay on the local file system for now. I recommend using the image upload handler as a reference.
Set an upload limit of 1 GB (1 << 30 bytes) using http.MaxBytesReader.
Extract the videoID from the URL path parameters and parse it as a UUID
Authenticate the user to get a userID
Get the video metadata from the database, if the user is not the video owner, return a http.StatusUnauthorized response
Parse the uploaded video file from the form data
Use (http.Request).FormFile with the key "video" to get a multipart.File in memory
Remember to defer closing the file with (os.File).Close - we don't want any memory leaks
Validate the uploaded file to ensure it's an MP4 video
Use mime.ParseMediaType and "video/mp4" as the MIME type
Save the uploaded file to a temporary file on disk.
Use os.CreateTemp to create a temporary file. I passed in an empty string for the directory to use the system default, and the name "tubely-upload.mp4" (but you can use whatever you want)
defer remove the temp file with os.Remove
defer close the temp file (defer is LIFO, so it will close before the remove)
io.Copy the contents over from the wire to the temp file
Reset the tempFile's file pointer to the beginning with .Seek(0, io.SeekStart) - this will allow us to read the file again from the beginning
Put the object into S3 using PutObject. You'll need to provide:
The bucket name
The file key. Use the same <random-32-byte-hex>.ext format as the key. e.g. 1a2b3c4d5e6f7890abcd1234ef567890.mp4
The file contents (body). The temp file is an os.File which implements io.Reader
Content type, which is the MIME type of the file.
Update the VideoURL of the video record in the database with the S3 bucket and key. S3 URLs are in the format https://<bucket-name>.s3.<region>.amazonaws.com/<key>. Make sure you use the correct region and bucket name!
Restart your server and test the handler by uploading the boots-video-vertical.mp4 file. Make sure that:
The video is correctly uploaded to your S3 bucket.
The video_url in your database is updated with the S3 bucket and key (and thus shows up in the web UI)


# 4.1 Object Storage

If you squint really hard, it feels like S3 is a file system in the cloud... but it's not. It's technically an object storage system - which is not quite the same thing.

Click to hide video

Traditional File Storage
"File storage" is what you're already familiar with:

Files are stored in a hierarchy of directories
A file's system-level metadata (like timestamp and permissions) is managed by the file system, not the file itself
File storage is great for single-machine-use (like your laptop), but it doesn't distribute well across many servers. It's optimized for low-latency access to a small number of files on a single machine.

Object Storage
Object storage is designed to be more scalable, available, and durable than file storage because it can be easily distributed across many machines:

Objects are stored in a flat namespace (no directories)
An object's metadata is stored with the object itself
Assignment
Open the video object we uploaded in the last lesson in the S3 console and look at the "Object overview" - you should see a bunch of metadata about the object:
Size
Type
Entity tag
etc.
Use the AWS CLI to read the metadata of your uploaded vertical video object, and redirect the output to /tmp/object_metadata.txt
aws s3api head-object --bucket BUCKET_NAME --key OBJECT_KEY > /tmp/object_metadata.txt

Replace BUCKET_NAME and OBJECT_KEY with the name of your bucket and the key of your object



# 4.2 File System Illusion

Remember how I mentioned that S3's "object storage" doesn't support directories? Well, that's true, but there's some trickery involved that makes it feel like it does.

Directories are really great for organizing stuff. Storing everything in one giant bucket makes a big hard-to-manage mess. So, S3 makes your objects feel like they're in directories, even though they're not.

It's Just Prefixes
Keys inside of a bucket are just strings. And strings can have slashes, right? Right.

If you upload an object to S3 with the key users/john/profile.jpg, we can kind of pretend that the object is in a directory called users and a subdirectory called john. Not only that, but the S3 API actually provides tools that allow this illusion to thrive.

Let's say I create some objects with keys:

users/dan/profile.jpg
users/dan/friends.jpg
users/lane/profile.jpg
users/lane/friends.jpg
people/matt/profile.jpg
Then I can use the S3 API to list all the objects with the key prefix users/lane. It returns:

users/lane/profile.jpg
users/lane/friends.jpg
or just everything with the prefix "users":

users/dan/profile.jpg
users/dan/friends.jpg
users/lane/profile.jpg
users/lane/friends.jpg
It feels like a hierarchy, without all the technical overhead of actually creating directories.

Assignment
Manually create a directory (folder) called "backups" in your bucket
Upload all the files from your samples directory there for safe keeping
Use the AWS CLI to list the files in each directory:
aws s3 ls s3://YOURBUCKET/backups/

Replace YOURBUCKET with the name of your bucket.

Do it again, but redirect the output to /tmp/s3_listing.txt:
aws s3 ls s3://YOURBUCKET/backups/ > /tmp/s3_listing.txt

# 4.3 Dynamic Path

Although directories are an illusion in S3, they're still useful due to the prefix filtering capabilities of the S3 API. There are a lot of common strategies for organizing objects in S3, but the most important rule is:

Organization matters.
Schema architecture matters in a SQL database, and prefix architecture matters in S3. We always want to group objects in a way that makes sense for our case, because often we'll want to operate on a group of objects at once.

For example, pretend you do the naive thing and upload all your images to the root of your bucket. What happens if...

you want to delete all the images for a specific user?
a feature changed and you need to resize all the images it uses?
you want to change the permissions of all the images associated with a specific organization?
If you don't have any prefixes (directories) to group objects, you might find yourself iterating over every object in the bucket to find the ones you care about. That's slow and expensive.

Assignment
Tubely's software architect has decided on the following prefix "schema" for our video uploads:

landscape (16:9 aspect ratio)
portrait (9:16 aspect ratio)
other (everything else)
Install ffmpeg (which will also install ffprobe):

# mac

brew install ffmpeg

# linux

sudo apt install ffmpeg

Ensure that the installation worked and the commands are available in your PATH:
ffprobe -version
ffmpeg -version

Use ffprobe manually to get the aspect ratio of a video file.
ffprobe -v error -print_format json -show_streams PATH_TO_VIDEO

You should see a streams array containing information about the video. We care about the width and height fields of the first stream.

Create a function getVideoAspectRatio(filePath string) (string, error) that takes a file path and returns the aspect ratio as a string.
It should use exec.Command to run the same ffprobe command as above. In this case, the command is ffprobe and the arguments are -v, error, -print_format, json, -show_streams, and the file path.
Set the resulting exec.Cmd's Stdout field to a pointer to a new bytes.Buffer.
.Run() the command
Unmarshal the stdout of the command from the buffer's .Bytes into a JSON struct so that you can get the width and height fields.
I did a bit of math to determine the ratio, then returned one of three strings: 16:9, 9:16, or other.
Aspect ratios might be slightly off due to rounding errors. You can use a tolerance range (or just use integer division and call it a day).
Update the handlerUploadVideo to get the aspect ratio of the video file from the temporary file once it's saved to disk. Depending on the aspect ratio, add a "landscape", "portrait", or "other" prefix to the key before uploading it to S3.
Test your code by using both the provided horizontal (16:9) and portrait (9:16) videos. You should see the correct prefix in the S3 bucket.
Delete all the videos from your <admin@tubely.com> account, and recreate them in this order (the tests retrieve videos sorted by most recent first):
Vertical video:
Title: "Boots Vertical"
Description: "A vertical video of boots"
Use the vertical video
Horizontal video:
Title: "Boots Horizontal"
Description: "A horizontal video of boots"
Use the horizontal video (this one might take a bit to upload)











