# Sensible-Public-GCS
A simple proxy for Google Cloud Storage buckets with sensible rate limiting.

<br>

# Why?
I've had a few times now where I've needed to host large files that shouldn't really go directly into Git and therefore I can't use GitHub Pages. I realised that Google Cloud Storage provides relatively cheap storage... but doesn't really do any rate limiting... So after several iterations of the idea, I made this.

<br>

# Who Should Use This?
This is mostly only designed for small scale hosting as it takes a relatively aggressive and naive approach, which could cause frustration and confusion for users. It's mainly designed for larger files and videos as Cloudflare doesn't allow video and Cloudflare Pages has a file size limit. If you have more traffic, a paid service like Cloudflare Stream is probably more suitible.

**Note**: You shouldn't use this to host HTML files as to access files proxied by this server, you need to prefix your request URLs with `/v1/object`. I'd also advise against using this to host JavaScript and CSS files as the server has some [limitations](#limitations).

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
 * `DAILY_EGRESS_PER_USER`: The daily egress limit for each user in bytes. **Default**: `500,000,000` (500MB).
 * `MAX_TOTAL_EGRESS`: The maximum total egress for the server per month in bytes. Note that this can be exceeded as sending 503s still uses egress. **Default**: `15,000,000,000` (15GB).
 * `MEASURE_TOTAL_EGRESS_FROM_ZERO`: If the total egress should be measured from 0 or should instead continue from what's been used that month. **Default**: `true`.
 * `MAX_TOTAL_REQUESTS`: The maximum total requests per month. **Default**: `50,000`.
 * `IS_PROXY_TEST`: When set to `true`, most endpoints are unregistered and a `/v1/headers` endpoint is registered.

<br>

# Setup
## Google Cloud Storage
First create a Google Cloud account and set up billing. I'd highly recommend also setting up billing alerts, just note that it won't shut anything down automatically.

Then create a new project. I'd suggest calling it "Public". And enable billing for the project.

Go to Cloud Storage, create a new bucket and call it something like `<-- your name -->-public-static`. Select the single region that's closest to where your server will be. Use the standard class. Leave public access protection on and decide if you want any backups. Click create, then upload your files.

Enable the Cloud Storage and Cloud Monitoring APIs.

Then go to IAM and Admin, go to roles and click "create role". Call the first one something like "Public Cloud Storage (custom)" and set the launch stage to General Availability. Click add permissions and give it `storage.objects.get` and `storage.objects.list`. Click create and repeat with another. Call it something like "Basic Monitoring Viewer (custom)" and give it the `monitoring.timeSeries.list` permission.

Then go to service accounts and click create. Give it a name like "Public-Static-GCS" then give it those 2 roles in the "Custom" section at the top. They might take a few minutes to appear. Click done.

Click on the email of the service account, go to "Keys" -> Add key -> Create new key. Choose JSON and click create. Open the downloaded file, copy the contents and open the JavaScript console in a private browser tab. Use this code to minify it:

```js
console.log(JSON.stringify(<-- paste the contents here, not in quotes -->))
```

Copy the result and store it somewhere safe for use in a minute.

## Server Setup
This server should be able to be hosted on most providers but note that currently, restarting the server will reset the usage counts, so it needs to be kept running. To keep to the spirit of this being cheap, I'd suggest:

 * [Fly.io](https://fly.io/docs/about/pricing/): Might be the best option but I haven't tried it yet.
 * [Railway](https://railway.app/pricing): Can be cheaper as you don't pay for idle resources, but now has a $5 per month minimum. Good option if you host enough there.
 * [Google App Engine](https://cloud.google.com/appengine/pricing): Only a good option if you stay within the free tier.

<br>

**Note**: The exact instructions will depend on your provider, so this part will be more of a guide than a tutorial.

First, upload this server's source code to your provider. Generally you can have it clone the Git repository, but otherwise you can download this as a zip and upload it. Your provider should detect the Dockerfile and build it.

Then set the environment variables by following the [Options](#options) section. I'd advise reviewing every option, but the required ones to set are:

 * `GIN_MODE`
 * `GCS_BUCKET_NAME`
 * `GCP_PROJECT_NAME`
 * `GCS_KEY`

Like with Google Cloud, I'd also highly recommend setting up some billing alerts and a hard limit if your provider supports it.

### Trusted Proxies
Generally when you deploy something to a cloud provider, the requests will be proxied and so will appear to come from the same few IP addresses. This creates issues with the rate limiting. To solve this, your provider will typically add a HTTP header to indicate the IP address of the client. For the following providers, that header is:

 * Google App Engine: `X-Appengine-Remote-Addr`
 * Railway: `X-Envoy-External-Address`

---

If your provider isn't listed there, you'll need to use this server's proxy test mode to find it. To do this, set the environment variable `IS_PROXY_TEST` to `true` and deploy the server. Then go to this URL in your browser: `<-- your server's origin -->/v1/ip`.

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
			"<-- header name to try -->": "<-- something that's not your IP like 1.1.1.1 -->"
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

**Now that you've got this HTTP header name**, set the environment variable `PROXY_ORIGINAL_IP_HEADER_NAME` to it. Then restart the server and ensure `<your server's origin>/v1/ip` returns your IP.

<br>

# REST API
Once you've got your server running, you'll be able to access a few different endpoints:

**GET**
 * **`/v1/object/*path`**: Proxies the request to Google Cloud Storage if the user's limits allow it.
 * `/v1/remaining/egress`: Returns a JSON object containing information about the user's remaining egress. It has a `"used"` and a `"remaining"` property, both measured in bytes.
 * `/v1/health`: Returns a 200 if the server is running.
 * `/v1/ip`: Sends the client's IP as a plain text string.
 * `/v1/headers` (only when `IS_PROXY_TEST` is `true`): Sends a JSON object with a `"headers"` property, which contains the HTTP headers of the request the server received as an object.


# Limitations
The server's main limitation is that it can only start to fetch one GCS object per user at a time. When a user requests an object, a lock is put on the user's data until the body of the underlying GCS request starts to be received by the server. This is fine for video streaming, which is the main use-case of this server, but will result in slightly reduced performance when objects are requested in parralel.

The rate limiting is rather naive as it only limits users on a per day basis, rather than having some sort of exponential backoff system. The overall limit is also just a hard cap so the server doesn't try to spread out its resource usage over the month. This means if there is some sort of DDoS attack (as opposed to coming from a single IP), the server will just shut down for the rest of the month unless your hosting provider detects and blocks the requests.

The server doesn't store any data, so restarts and crashes will reset the usage metrics.

# Privacy
The server doesn't store or send off any user data. However, Gin logs each request to the standard output, including the user's IP, so you should periodically delete old logs. My code will also log IPs in a couple of scenarios like when there's a new user or one is forgotten.

While this information isn't persistently stored, some user information is kept in memory:

 * IP address
 * Egress used since it was reset
 * When the egress was reset (essentially)

A user's egress usage is reset to zero 24 hours after their first request. When a user's egress usage is zero, they're eligible to be forgotten. These users are checked for every 12 hours and forgotten. This all means it takes a maximum of 36 hours for a user to be forgotten after they've made their last request.