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
