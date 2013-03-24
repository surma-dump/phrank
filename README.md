`phrank` is a generic reverse proxy born out of the desire to build
pure client-side apps accessing third party APIs. It is designed to be
easily cloud-deployable.

# Configuration
All configuration is done using command line flags.

## Ressources
Ressources can be:

* Another HTTP URL (e.g. `http(s)://<Domain>`)
* A [S3][3] Bucket (e.g. `s3://<Key>:<Secret>@<S3 Endpoint>/<Bucket>/<prefix>`)
* Local, static content (e.g. `file://<path>`)

To define a map, just specifiy a `--map` flag for each ressource that you use:

	$ phrank --map "/images => s3://key:secret@s3.amazonaws.com/mymediabucket/images" \
	         --map "/ => file://./static"

## Caching
The caching parameters specifies the duration for which static content is kept in
memory to avoid unnecessary requests.

Only S3 and local content is cached.

# Usage & Deployment
The usual use-case is to fork this repository and deploy it to [Heroku][1],
[dotCloud][2] or any other PaaS that supports Go.

You can either add your static content to the repository and define a `file` map
or put it in a [S3][3] bucket and pick a rather large caching time.

[1]: http://heroku.com
[2]: http://dotcloud.com
[3]: http://aws.amazon.com/s3

