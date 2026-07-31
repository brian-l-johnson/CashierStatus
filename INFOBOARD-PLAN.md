# Info Board — implementation plan for CashierStatus

Paste this into a Claude Code session running in `/Users/bj/dev/CashierStatus`.

## Goal

Add a rotating informational board (rules / notices) to the existing CashierStatus
app. A display Pi shows it by pointing its `kiosk_url` at `/infoboard`. Content is
managed from a new admin page. Each note is a message plus an optional image.

## Why this repo and not a new app

The display side already exists. `merch-app`'s `cmd/merch-browser` is a Chromium
kiosk launcher that reads `kiosk_url` from `/etc/merch/merch-app.env`, polls that
file, and relaunches when the value changes (`cmd/merch-browser/main.go:104-109`).
So an info board node is just an existing display Pi with:

```
kiosk_url=http://<cashierstatus-host>/infoboard
```

No new binary, repo, systemd unit, CI signing, or provisioning. The node keeps
heartbeating into the "Merch Status" column via `pkg/checkin` like every other
station.

CashierStatus already supplies everything the server side needs: Gin + GORM +
SQLite with `AutoMigrate`, role-based auth (`admin` / `view` / `update`), an
admin-CRUD page pattern, and a nav menu.

## Existing conventions to follow

Read these first — the new code should look like them:

- `models/cashier.go` — model shape (embedded `gorm.Model` with `json:"-"`, plus
  an explicit exported field set so it appears in JSON)
- `models/db.go` — `Init()` calls `db.AutoMigrate(...)` per model; `GetDB()`
  returns the handle; `GenerateRandomString(n)` is already there and is the right
  thing to use for upload filenames
- `controllers/cashiers.go` — controller struct with methods, `c.IndentedJSON`,
  `gin.H{"status": ..., "message": ...}` envelopes, swaggo doc comments
- `server/router.go` — route registration, `middleware.Authorize("admin")` for
  JSON writes, `middleware.AuthorizeHTML("admin")` for pages
- `templates/cashierscreate.html` — admin CRUD page: Bootstrap from
  `/static/vendor/bootstrap.min.css`, `{{ template "menu.html" . }}`, plain
  `fetch()` calls, results logged into a `#results` div
- `templates/main.html` — the existing board page: no menu chrome, hidden nav
  revealed by pressing `m` (Esc hides), `static/common.css`

Note `router.LoadHTMLGlob("templates/*")` — new templates load automatically.
Module path is `github.com/brian-l-johnson/CashierStatusBoard/v2`.

## 1. Model — `models/note.go`

```go
package models

import "gorm.io/gorm"

// Note is one rotating message on the info board. ImageURL is always either
// empty or a server-local "/uploads/..." path — never an external URL. See
// the upload handler for why.
type Note struct {
	gorm.Model `json:"-"`
	ID         uint   `json:"id"`
	Message    string `json:"message"`
	ImageURL   string `json:"image_url"`
	Position   int    `json:"position"`
	Active     bool   `json:"active"`
	DwellSec   int    `json:"dwell_sec"` // 0 = use board default
}
```

Register it in `models.Init()` alongside the others:

```go
db.AutoMigrate(&Note{})
```

## 2. Uploads — writable path

`router.Static("/static", "./static")` serves files baked into the image, so
uploads must not go there or they vanish on redeploy.

Add an upload dir resolved from env, defaulting to `./uploads`, created with
`os.MkdirAll` at startup, and served read-only:

```go
uploadDir := os.Getenv("UPLOAD_DIR")
if uploadDir == "" {
    uploadDir = "./uploads"
}
os.MkdirAll(uploadDir, 0o755)
router.Static("/uploads", uploadDir)
```

**Deployment — resolved.** The `Dockerfile` already solves this for the database
and the same mechanism covers uploads. It pre-creates `/data` in the build stage
and copies it in `--chown=nonroot:nonroot` (distroless has no shell to `mkdir` in
the final stage), then declares `VOLUME ["/data"]` and `ENV DB_PATH=/data/...`.
So add one line next to `DB_PATH`:

```dockerfile
ENV UPLOAD_DIR=/data/uploads
```

`os.MkdirAll` at startup then succeeds as uid 65532 inside the mounted volume.

Also add `uploads/` to both `.gitignore` and `.dockerignore` — neither excludes
it today, so local test uploads would otherwise get committed and baked into the
image.

**Note the exposure:** `/uploads` is served unauthenticated, because the display
Pis have no session and must be able to load the images. Anything uploaded is
readable by anyone who can reach the host. That is the correct design here, but
it means the info board is not a place for staff-only material.

## 3. Endpoints — `controllers/notes.go`, registered in `server/router.go`

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/notes` | none | Active notes, ordered by `Position` — what the board polls |
| GET | `/notes/all` | `admin` | All notes including inactive, for the admin table |
| POST | `/notes` | `admin` | Create |
| PUT | `/notes/:nid` | `admin` | Update |
| DELETE | `/notes/:nid` | `admin` | Delete |
| POST | `/notes/upload` | `admin` | Image upload, returns `{"image_url": "/uploads/..."}` |
| GET | `/infoboard` | none | The display page |
| GET | `/notes/manage` | `admin` (HTML) | Admin page |

Follow the auth style already in `router.go`: `middleware.Authorize("admin")` for
the JSON writes, `middleware.AuthorizeHTML("admin")` for `/notes/manage`.

`GET /notes` must be unauthenticated — display Pis have no session.

**Route shape:** `GET /notes/all` and `GET /notes/manage` coexist with
`PUT|DELETE /notes/:nid` only because they are different methods. Gin's router
will panic *at startup* if anyone later adds a `GET /notes/:nid`. Leave a comment
at the registration site saying so.

### ETag on `GET /notes`

The board polls this on a timer. Serve an ETag so polls are cheap and so the
client can tell "nothing changed" from "changed".

Do **not** use `c.IndentedJSON` here, unlike every other endpoint in this repo.
It would re-serialize with different whitespace than whatever you hashed, and the
ETag would never match the body. Marshal exactly once and write those same bytes:

```go
b, err := json.Marshal(notes)
// ...
sum := sha256.Sum256(b)
etag := `"` + hex.EncodeToString(sum[:]) + `"`
c.Header("ETag", etag) // must be set on the 304 path too, not just the 200
if c.GetHeader("If-None-Match") == etag {
    c.Status(http.StatusNotModified)
    return
}
c.Data(http.StatusOK, "application/json; charset=utf-8", b)
```

### Ordering

Order by `position asc, id asc`, never `position` alone. Nothing enforces unique
positions and the up/down swap buttons make collisions easy; tied rows would come
back in whatever order SQLite felt like, so the ETag would flap between two
otherwise identical polls and every board would re-fetch on every tick.

### Validation on write

- `Message`: required, trim, reject over ~500 chars.
- `ImageURL`: must be empty **or** match `^/uploads/[A-Za-z0-9]+\.(png|jpg|jpeg|gif|webp)$`.
  Reject anything else. This is the whole defense against pointing kiosks at
  arbitrary hosts and against `javascript:` / `data:` URLs — do not skip it, and
  do not accept an operator-supplied external URL "just this once".
- `DwellSec`: 0, or clamp to 5..300.

### Upload handler

- Cap the body: `c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)`.
- Read the first 512 bytes and pass to `http.DetectContentType`. Accept only
  `image/png`, `image/jpeg`, `image/gif`, `image/webp`. Do **not** trust the
  client's `Content-Type` header or the uploaded filename.
- Build the filename yourself: `models.GenerateRandomString(16)` plus the
  extension implied by the *detected* type. Never join a user-supplied filename
  into a path. Note the real signature is `GenerateRandomString(n) (string, error)`
  — handle the error rather than discarding it; this is the security-critical path.
  Its charset is `[0-9A-Za-z]`, which is what makes the `ImageURL` regex below
  able to be that strict.
- Write into `uploadDir`, return `{"image_url": "/uploads/<name>"}`.

Deleting a note deliberately does **not** delete its uploaded image. `gorm.Model`
makes deletes soft, so the file simply becomes unreferenced. At a few notes per
event that leak is irrelevant; revisit only if this ever grows an upload history.

## 4. Display page — `templates/infoboard.html`

Full-screen, dark, sized for a TV at a distance. Model the chrome on
`main.html` (no nav bar; keep the `m` / Esc hidden-menu behavior if you want a
way to get out of it on the floor).

Behavior, in priority order — these three are the ones that actually go wrong:

1. **Do not restart the rotation on poll.** Keep the fetched list in a staging
   variable and swap it in only at the next slide boundary. A naive
   implementation re-renders on every poll and the board sits on slide 1 forever.
   If the current index is past the end of the new list, wrap to 0.
2. **Set text with `textContent`, never `innerHTML`.** The message comes from a
   form, so `innerHTML` makes the admin page a stored-XSS sink aimed at a kiosk
   browser. Same for building the image element — set `img.src` to the validated
   path, don't template a string.
3. **Cache the last good list in `localStorage`** and render from it on load
   before the first fetch resolves. Conference WiFi drops; a board that goes
   blank during an outage is worse than one showing a stale list. On fetch
   failure, keep rotating what's already there and retry on the next tick.

Also:

- Poll interval 60s. Rotation 20s default, per-note override from `DwellSec`,
  global default overridable via a `data-` attribute or query param if you want
  to tune it on the floor without a rebuild.
- Preload the next slide's image before advancing, otherwise you get a flash of
  empty layout on each transition.
- Set `img.onerror` to hide the image element. The localStorage path restores
  text fine with the server down, but a cached `/uploads/...` src that no longer
  loads renders a broken-image icon at TV scale, which looks worse than no image.
- Reuse the brand palette already defined at the top of `static/common.css`
  (`--color-navy`, `--color-mint`, …) rather than inventing new colors.
- If the list is empty, show a neutral placeholder rather than a blank screen.
- Progress dots or a thin timer bar so it's visibly alive.
- Handle a single-note list without dividing by zero or thrashing the DOM.

## 5. Admin page — `templates/notes.html`

Copy the structure of `cashierscreate.html`.

The `/notes/manage` handler must put `user` and `roles` into the `gin.H`, like
every other admin route does. `menu.html` calls `{{if .roles | isAdmin}}` and the
registered `isAdmin` func takes a `string`, so omitting `roles` fails the render.

- Table of existing notes: message, thumbnail, position, active toggle, edit,
  delete. Load from `GET /notes/all`.
- Create/edit form: message textarea, image file input, dwell seconds, position,
  active checkbox.
- Image flow: upload first via `POST /notes/upload`, take the returned
  `image_url`, then include it in the note `POST`/`PUT`.
- Reordering: simplest workable version is an integer position field plus
  up/down buttons that swap positions. Drag-and-drop is not worth it here.
- A "preview" link to `/infoboard` in a new tab.

## 6. Menu — `templates/menu.html`

Add an entry inside the existing `{{if .roles | isAdmin}}` block:

```html
<li class="nav-item">
    <a class="nav-link" href="/notes/manage">Info Board</a>
</li>
```

## 7. Swagger

The repo generates docs from swaggo comments (`docs/docs.go`, `/swagger/*any`).
Add the same style of annotation block to each new controller method as in
`controllers/cashiers.go`.

There is no Makefile and no CI step for this — `docs/docs.go` is committed and
refreshed by hand with a locally installed `swag init`. So the annotations are
inert until someone runs that. Write them anyway (they're the convention), and
regenerate only if `swag` is already on the path.

## Verify before finishing

- `go build ./...` clean.
- `go test ./...` clean. The repo has no tests today and this plan does not
  change that wholesale, with one exception: the `ImageURL` validator is a pure
  function that is the entire defense against pointing a kiosk at an arbitrary
  host, so it gets a table test.
- `GET /notes` returns `[]` on a fresh DB and does **not** require auth.
- Second `GET /notes` with `If-None-Match` returns `304`.
- Writes rejected without an admin session; accepted with one.
- Upload of a non-image (e.g. rename a `.txt` to `.png`) is rejected by content
  sniffing.
- A note with `image_url` set to `https://evil.example/x.png` or
  `javascript:alert(1)` is rejected by the write validator.
- A note whose message is `<img src=x onerror=alert(1)>` renders as literal text
  on `/infoboard`.
- Board keeps rotating with the server stopped, and picks up changes within one
  poll interval of it coming back.
- Adding a note does not reset the board to slide 1 mid-cycle.

## Notes / open items

- ~~I did not read the `Dockerfile`, `.env`, or `middleware/authorize.go`~~ —
  all three checked. The upload volume story is resolved in §2. `AuthorizeHTML`
  does redirect to `API_BASE_URL + /login?from=...` when there is no session
  (`middleware/authorize.go:31-34`); note it returns a bare `401` rather than a
  redirect when a user *is* logged in but lacks the role.
- The `cmd/merch-browser` kiosk-relaunch behavior lives in another repo and was
  not verifiable from here. Confirmed correct by the repo owner, 2026-07-31.
- The existing `Cashier` model embeds `gorm.Model` *and* redeclares `ID`. The
  `Note` model above follows that same pattern for consistency; if you'd rather
  drop the embed and use explicit `gorm:"primarykey"` plus `json` tags, that's
  cleaner, just do it deliberately.
- Deliberately **not** reusing the SSE broadcaster
  (`/cashiers/getupdate-sse`). It's typed to the cashier `Message` struct and
  would need widening; notes change a few times a day, the board already runs a
  rotation timer, and a poll recovers from a network blip without connection
  bookkeeping.
