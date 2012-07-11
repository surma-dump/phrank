`phrank` is a reverse proxy with SSL termination.

# Configuration

## Apps

For every backend, app or whatever you want to call it, there needs to be a config file in `phrank.d` like this:

	{
		"Domain": "app.domain.com",
		"AddForwardHeader": false,
		"Address": "localhost:9001"
	}

Requests made to `phrank` with the `Host` header set to `Domain` are forwarded to `Address`. If the `AddForwardHeader` field is true, an additional header `X-Forwarded-For` will be injected into the request to the backend containing the address of the original client.

Config file names have to end in `.app`.

## SSL

Right now the SSL key and certificate are assumed to be in the configuration directory (usually `phrank.d`) and to be called `key.pem` and `cert.pem`. If one of them is not found, no HTTPS server is started.

## Reloading

`phrank` can be forced to reread all configurations (not the certificates, though) by sending it the `SIGUSR1` signal:

	killall -USR1 phrank
