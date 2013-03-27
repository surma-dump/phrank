`phrank` is a generic reverse proxy born out of the desire to build
pure client-side apps accessing third party APIs. It is designed to be
easily cloud-deployable.

# Configuration
All configuration is done using command line flags.

## Resources
Resources can be:

* Another HTTP URL (i.e. `http(s)://<Domain>/<Path>`)
* A [S3][3] Bucket (i.e. `s3://<Key>:<Secret>@<S3 Endpoint>/<Bucket>/<Prefix>`)
* Local, static content (i.e. `file://<Path>`)

To define a map, just specifiy a `--map` flag for each resource that you use:

	$ phrank --map "/images => s3://key:secret@s3.amazonaws.com/mymediabucket/images" \
	         --map "/ => file://./static"

## Caching
The caching parameters specifies the duration for which static content is kept in
memory. Only S3 and local content is cached.

# Usage & Deployment
My usual use-case is to fork this repository and deploy it to [Heroku][1],
[dotCloud][2] or any other PaaS that supports Go.

Examples will follow...

[1]: http://heroku.com
[2]: http://dotcloud.com
[3]: http://aws.amazon.com/s3
---
Version 1.0.1
