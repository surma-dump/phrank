`phrank` is a generic reverse proxy born out of the desire to build
pure client-side apps accessing third party APIs. It is designed to be
easily cloud-deployable.

# Configuration
All configuration is done using command line flags.

## Resources
Resources can be:

* Another HTTP URL (i.e. `http(s)://<Domain>/<Path>`)
* Local, static content (i.e. `file://<Path>`)

To define a map, just specifiy a `--map` flag for each resource that you use:

	$ phrank --map "/api => https://api.twitter.com" \
	         --map "/ => file://./static"

# Usage & Deployment
My usual use-case is to fork this repository and deploy it to [Heroku][1],
[dotCloud][2] or any other PaaS that supports Go.

[1]: http://heroku.com
[2]: http://dotcloud.com
---
Version 2.0.0
