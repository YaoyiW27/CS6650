# CS6650 Midterm — Hummingbird API Debugging Report

## Ticket #1 — Wrong Port (Easy)

**Bug:** `server.js`, line 35 
**Problem:** The port was set using `process.env.APP_PORT` with no fallback value. If the `APP_PORT` environment variable is not set, `port` becomes `undefined`. In Node.js, calling `app.listen(undefined)` does not crash — instead, it starts the server on a random OS-assigned port instead of the expected port 9000. The log would misleadingly say "listening on port undefined."

**Code Diff:**
```diff
# server.js:35
- const port = process.env.APP_PORT;
+ const port = process.env.APP_PORT || 9000;
```

**Verification:** After deploying, `curl http://<alb-dns>/health` returns `{"status":"ok","service":"hummingbird"}`, confirming the server is listening on the correct port.

---

## Ticket #2 — Missing Width in Metadata (Easy)

**Bug:** `clients/dynamodb.js`, line 78–84 (getMedia function)
**Problem:** The `createMedia` function saves `width` to DynamoDB along with other fields (size, name, mimetype, status). However, the `getMedia` function's return object only maps back `mediaId`, `size`, `name`, `mimetype`, and `status` — it omits `width` entirely. So even though `width` is stored in DynamoDB, the GET endpoint never returns it.

**Code Diff:**
```diff
# clients/dynamodb.js:84 (inside getMedia return object)
  status: Item.status.S,
+ width: Number(Item.width.N),
```

**Verification:** After uploading with `?width=500`, `GET /v1/media/<mediaId>` now returns `"width": 500` in the response.

---

## Ticket #3 — Broken Redirect URL (Intermediate)

**Bug:** `controllers/media.js`, line 111
**Problem:** The `Location` header in the 202 response was built using `req.hostname`, which in Express returns just the hostname without a protocol prefix (e.g., `hummingbird-alb-xxx.elb.amazonaws.com`). This produces an invalid URL like `hummingbird-alb-xxx.elb.amazonaws.com/v1/media/abc/status` — missing `http://`. Clients cannot follow this redirect because it's not a valid URL.

**Code Diff:**
```diff
# controllers/media.js:111
- res.set('Location', `${req.hostname}/v1/media/${mediaId}/status`);
+ res.set('Location', `http://${req.get('host')}/v1/media/${mediaId}/status`);
```

**Verification:** `curl -i /download` now returns a `Location` header with a proper `http://` prefix.

---

## Ticket #4 — Download Never Redirects (Intermediate)

**Bug:** `controllers/media.js`, line 108
**Problem:** The condition that decides whether to return a 202 (retry) vs 302 (redirect) was checking `media.status !== MEDIA_STATUS.PROCESSING`. This means the 202 response fires when status equals anything *other than* PROCESSING — including COMPLETE. The logic is inverted: it should check `media.status !== MEDIA_STATUS.COMPLETE`, so that only non-COMPLETE statuses trigger the 202 retry, and COMPLETE triggers the 302 redirect.

**Code Diff:**
```diff
# controllers/media.js:108
- if (media.status !== MEDIA_STATUS.PROCESSING) {
+ if (media.status !== MEDIA_STATUS.COMPLETE) {
```

**Verification:** After triggering resize and waiting for COMPLETE status, `curl -i /download` now returns `HTTP/1.1 302 Found` with a presigned S3 URL redirect.

---

## Ticket #5 (Bonus) — Status Never Updates

**Bug:** `clients/dynamodb.js`, lines 154 and 166 (setMediaStatus function)
**Problem:** The `setMediaStatus` function uses a DynamoDB sort key `SK: { S: 'metadata' }` (lowercase), while `createMedia` and `getMedia` both use `SK: { S: 'METADATA' }` (uppercase). In DynamoDB, the partition key + sort key combination uniquely identifies an item. Because of the casing mismatch, `setMediaStatus` is not updating the existing item — instead, it silently creates a *new* DynamoDB item with the lowercase sort key. The original item's status remains `PENDING` forever. There are zero errors in the logs because DynamoDB's `UpdateItem` creates a new item when the key doesn't exist (upsert behavior) if there's no `ConditionExpression`.

**Code Diff:**
```diff
# clients/dynamodb.js:154
- SK: { S: 'metadata' },
+ SK: { S: 'METADATA' },

# clients/dynamodb.js:166 (log message)
- SK: { S: 'metadata' },
+ SK: { S: 'METADATA' },
```

**Verification:** After triggering resize, `GET /v1/media/<mediaId>/status` now correctly returns `{"status":"COMPLETE"}` instead of being stuck on `PENDING`.

---

## Summary of All Fixes

| Ticket | Bug | File & Line | Fix |
|--------|-----|-------------|-----|
| #1 | Port fallback | server.js:35 | Added `\|\| 9000` default |
| #2 | Missing width | clients/dynamodb.js:84 | Added `width: Number(Item.width.N)` |
| #3 | Broken redirect URL | controllers/media.js:111 | Added `http://` prefix, used `req.get('host')` |
| #4 | Download never redirects | controllers/media.js:108 | Changed condition to check `MEDIA_STATUS.COMPLETE` |
| #5 | Status never updates | clients/dynamodb.js:154,166 | Changed sort key from `'metadata'` to `'METADATA'` |
