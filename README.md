# Sensible-Public-GCS
A simple proxy for Google Cloud Storage buckets with sensible rate limiting.

<br>

# Why?
I've had a few times now where I've needed to host large files that shouldn't really go directly into Git and therefore I can't use GitHub Pages. I realised that Google Cloud Storage provides relatively cheap storage... but doesn't really do any rate limiting. So after several iterations of the idea, I made this.

<br>

# Who Should Use This?
This is mostly only designed for small scale hosting as it takes a relatively aggressive approach, which could cause frustration and confusion for users. It's mainly designed for larger files and videos as Cloudflare doesn't allow video and Cloudflare Pages has a file size limit. If you have more traffic, a paid service like Cloudflare Stream is probably more suitible.

<br>

# How Does it Work?
**Disclaimer**: As per the license terms, this software is provided "as is" and so I can't be held responsible for any unexpected costs due to the actual behaviour differing to what's described here. 

The server tracks the egress usage both per user (using their IP) and overall. When a user makes a request, it gets forwarded on to Google Cloud provided the user has at least 5MB left and the total limits haven't been reached. When the response is received from Google Cloud, the server ensures the `Content-Length` + 100KB is smaller than the user's remaining egress, otherwise the response is cancelled. The user's remaining egress is reduced by the size + 100KB, with a minimum of 5MB and can now make another request. If the response is cancelled, the user is refunded the size that was left (though this can be reduced by the 100KB overhead and 5MB minimum). Their bandwidth is reset every 24h but they might not be deleted for up to 12 hours after that.

The overall limits are somewhat similar. Before doing any other checks, the server ensures the maximum number of monthly requests hasn't yet been reached, otherwise it sends a 503. The overall egress is then capped in a similar way to how it is for users (5MB check then using the content length), but the monthly usage provided by Google Cloud is also used:

 * When the server checks if it has enough remaining egress, it uses a cautiously high figure for the total. This is the provisional egress (which starts at 0) plus what Google Cloud reports (which usually updates about 2 minutes after the request finishes).
 * The provisional egress is increased just before a request is sent to Google Cloud using the 5MB minimum, then corrected using the content length (plus the overhead and minimum).
 * When a request finishes, it's corrected using how much was sent to the client (plus the overhead and minimum).
 * 3 minutes after, Google Cloud will have updated and all the increases to the provisional egress from the request are undone.

<br>

# Options
This server is configured using environment variables and the `.env.local.keys`, `.env.local` and `.env` files (in that priority). The env files are mainly only used for development so you should instead use your cloud provider's system for setting environment variables to configure things. However, I'd suggest you look at the included `.env` file to ensure the defaults are sensible in your opinion.

Anyway, the options are:
 * `PORT`: The port for the server to listen on. It'll likely be overwritten by your cloud provider. **Default**: `8000`.
 * `CORS_ALLOWED_ORIGINS`: Which origins to allow to access your server. **Default**: `*`.
 * `GIN_MODE`: Controls some debug features, **set this to `"release"`** for deploys. **Default**: `"debug"`.
 * `PROXY_ORIGINAL_IP_HEADER_NAME`: See [Trusted Proxies](#trusted-proxies). **Default**: `""`.
 * `GCS_BUCKET_NAME`: The name of the Google Cloud Storage bucket to proxy. 
 * `GCP_PROJECT_NAME`: The ID of the Google Cloud project. Usually ends in a dash and a number.
 * `GCS_KEY`: The contents of your Google Cloud service account's key file.
 * `DAILY_EGRESS_PER_USER`: The daily egress limit for each user in bytes. **Default**: `500000000` (500MB).
 * `MAX_TOTAL_EGRESS`: The maximum total egress for the server per month in bytes. Note that this can be exceeded as sending 503s still uses egress. **Default**: `15,000,000,000` (15GB).
 * `MEASURE_TOTAL_EGRESS_FROM_ZERO`: If the total egress should be measured from 0 or should instead continue from what's been used that month. **Default**: `true`.
 * `MAX_TOTAL_REQUESTS`: The maximum total requests per month. **Default**: `50,000`.
 * `IS_PROXY_TEST`: When enabled, most endpoints are unregistered and a `/v1/headers` endpoint is registered.

<br>

# Setup
## Google Cloud Storage


## Trusted Proxies
Generally when you deploy something to a cloud provider, the requests will be proxied and will so appear to come from the same few IP addresses, which creates issues with the rate limiting. To solve this, your provider will typically add a HTTP header to indicate the IP address of the client. For the following providers, that header is:

 * Google App Engine: `X-Appengine-Remote-Addr`
 * Railway: `X-Envoy-External-Address`

---

If your provider isn't listed there, you'll need to use this server's proxy test mode to find it. To do this, set the environment variable `IS_PROXY_TEST` to `true` and deploy the server. Then go to this URL in your browser: `<your server's domain>/v1/ip`.

If the IP listed doesn't match the IP you get when you Google "what's my IP", it means the requests are being proxied but not being handled correctly. The IP should also be a local address starting with `192.168` (or an IPv6 equivalent).

To find the HTTP header with the client's IP, open the JavaScript console on that page from the server and run this code:

```js
(async () => {
	const res = await fetch("/v1/headers");
	console.log({...(await res.json())});
})();
```

Expand the object and you should get something like this:

```json
{
	"headers": {
		// ...
		"X-Envoy-External-Address": "42.42.42.42",
		"X-Forwarded-For": "42.42.42.42",
		// Maybe some more x-headers...
	}
}
```

In this example, the IP address is `42.42.42.42` and so the headers `X-Forwarded-For` and `X-Envoy-External-Address` are the ones to focus on. `X-Forwarded-For` is the most standard but can often be spoofed by clients. To check if a header's safe, modify and run this code to ensure the server is overwriting it:

```js
(async () => {
	const res = await fetch("/v1/headers", {
		headers: {
			"<header name to try>": "<something that's not your IP like 1.1.1.1>"
		}
	});
	console.log({...(await res.json())});
})();
```

If you get something like this, you'll need to try a different header:

```json
{
	"headers": {
		// ...
		"X-Envoy-External-Address": "42.42.42.42",
		"X-Forwarded-For": "1.1.1.1,42.42.42.42", // The value sent by the client got merged in
		// Maybe some more x-headers...
	}
}
```

Otherwise, if the header you tried wasn't affected by the client's modification, that's the header to use. Before continuing, make sure you disable the proxy test again by setting the environment variable `IS_PROXY_TEST` back to `false` or by not setting it, as otherwise the server won't work.

---

**Now that you've got this HTTP header**, set the environment variable `PROXY_ORIGINAL_IP_HEADER_NAME` to it. Then restart the server and ensure `<your server's domain>/v1/ip` returns your IP.