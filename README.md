# Sensible-Public-GCS
A simple proxy for Google Cloud Storage buckets with sensible rate limiting.

# Setup
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
TODO
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
TODO
```

Otherwise, if the header you tried wasn't affected by the client's modification, that's the header to use.

---

**Now that you've got this HTTP header**, set the environment variable `PROXY_ORIGINAL_IP_HEADER_NAME` to it.